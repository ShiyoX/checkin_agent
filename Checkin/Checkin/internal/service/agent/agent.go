package agent

import (
	"Checkin/internal/dao/query"
	ragSvc "Checkin/internal/service/rag"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type AgentService struct {
	client     *openai.Client
	model      string
	ragService *ragSvc.RAGService
}

func NewAgentService(cfg *viper.Viper) *AgentService {
	apiKey := cfg.GetString("llm.api_key")
	baseURL := cfg.GetString("llm.base_url")
	model := cfg.GetString("llm.model")

	if apiKey == "" {
		zap.L().Warn("LLM API Key is not configured, agent service will not work properly")
	}

	config := openai.DefaultConfig(apiKey)
	if baseURL != "" {
		config.BaseURL = baseURL
	}

	return &AgentService{
		client:     openai.NewClientWithConfig(config),
		model:      model,
		ragService: ragSvc.NewRAGService(cfg),
	}
}

// getUserCheckinInfo 查询用户指定月份的签到信息和总积分
func (s *AgentService) getUserCheckinInfo(ctx context.Context, userID int64, year int, month int) (string, error) {
	userPoint, err := query.UserPoint.WithContext(ctx).
		Where(query.UserPoint.UserID.Eq(userID)).
		First()

	pointsInfo := "无法获取积分信息"
	if err == nil && userPoint != nil {
		pointsInfo = fmt.Sprintf("当前可用积分: %d, 累计获得积分: %d", userPoint.Points, userPoint.PointsTotal)
	}

	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	endDate := startDate.AddDate(0, 1, -1)

	records, err := query.UserCheckinRecord.WithContext(ctx).
		Where(query.UserCheckinRecord.UserID.Eq(userID)).
		Where(query.UserCheckinRecord.CheckinDate.Between(startDate, endDate)).
		Find()

	if err != nil {
		return "", fmt.Errorf("获取签到记录失败: %v", err)
	}

	daysMap := make(map[int]int32)
	for _, record := range records {
		day := record.CheckinDate.Day()
		daysMap[day] = record.CheckinType
	}

	checkedDays := len(daysMap)
	result := fmt.Sprintf("【用户信息摘要】\n%s\n\n【%d年%d月签到详情】\n总天数: %d\n本月已签到天数: %d\n详细记录: ",
		pointsInfo, year, month, endDate.Day(), checkedDays)

	if checkedDays == 0 {
		result += "本月尚未签到。"
	} else {
		for i := 1; i <= endDate.Day(); i++ {
			if t, ok := daysMap[i]; ok {
				if t == 1 {
					result += fmt.Sprintf("%d日(正常签到), ", i)
				} else {
					result += fmt.Sprintf("%d日(补签), ", i)
				}
			}
		}
	}

	return result, nil
}

// searchKnowledgeBase 从用户的 RAG 知识库中检索信息
func (s *AgentService) searchKnowledgeBase(ctx context.Context, userID int64, query string) (string, error) {
	if s.ragService == nil {
		return "知识库服务未配置", nil
	}

	if !s.ragService.HasUserIndex(ctx, userID) {
		return "用户尚未上传知识库文件", nil
	}

	docs, err := s.ragService.Retrieve(ctx, userID, query)
	if err != nil {
		return "", fmt.Errorf("知识库检索失败: %v", err)
	}

	if len(docs) == 0 {
		return "知识库中未找到相关内容", nil
	}

	result := "【知识库检索结果】\n"
	for i, doc := range docs {
		result += fmt.Sprintf("[文档片段 %d]: %s\n\n", i+1, doc)
	}
	return result, nil
}

type checkinArgs struct {
	Year  int `json:"year"`
	Month int `json:"month"`
}

type knowledgeArgs struct {
	Query string `json:"query"`
}

// buildTools 构建工具列表，根据用户是否有知识库动态添加
func (s *AgentService) buildTools(ctx context.Context, userID int64) []openai.Tool {
	tools := []openai.Tool{
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "get_user_checkin_info",
				Description: "获取用户指定年月的签到记录和积分信息",
				Parameters: jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"year": {
							Type:        jsonschema.Integer,
							Description: "年份，例如 2024",
						},
						"month": {
							Type:        jsonschema.Integer,
							Description: "月份，1-12",
						},
					},
					Required: []string{"year", "month"},
				},
			},
		},
	}

	// 如果用户有知识库，添加知识库检索工具
	if s.ragService != nil && s.ragService.HasUserIndex(ctx, userID) {
		tools = append(tools, openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "search_knowledge_base",
				Description: "从用户上传的知识库文档中检索相关信息。当用户的问题可能与其上传的文档内容相关时使用此工具",
				Parameters: jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"query": {
							Type:        jsonschema.String,
							Description: "检索查询内容，应该是与用户问题相关的关键词或短语",
						},
					},
					Required: []string{"query"},
				},
			},
		})
	}

	return tools
}

// handleToolCall 处理单个工具调用
func (s *AgentService) handleToolCall(ctx context.Context, userID int64, toolCall openai.ToolCall, now time.Time) string {
	switch toolCall.Function.Name {
	case "get_user_checkin_info":
		var args checkinArgs
		if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
			zap.L().Error("解析签到工具参数失败", zap.Error(err))
			return "参数解析失败"
		}
		if args.Year == 0 {
			args.Year = now.Year()
		}
		if args.Month == 0 {
			args.Month = int(now.Month())
		}
		result, err := s.getUserCheckinInfo(ctx, userID, args.Year, args.Month)
		if err != nil {
			return fmt.Sprintf("获取数据失败: %v", err)
		}
		return result

	case "search_knowledge_base":
		var args knowledgeArgs
		if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
			zap.L().Error("解析知识库工具参数失败", zap.Error(err))
			return "参数解析失败"
		}
		result, err := s.searchKnowledgeBase(ctx, userID, args.Query)
		if err != nil {
			return fmt.Sprintf("知识库检索失败: %v", err)
		}
		return result

	default:
		return fmt.Sprintf("未知工具: %s", toolCall.Function.Name)
	}
}

// Chat 处理用户对话
func (s *AgentService) Chat(ctx context.Context, userID int64, message string) (string, error) {
	if s.client == nil || s.model == "" {
		return "系统未正确配置AI助手，请联系管理员。", nil
	}

	now := time.Now()

	hasKB := s.ragService != nil && s.ragService.HasUserIndex(ctx, userID)
	kbHint := ""
	if hasKB {
		kbHint = "\n你还可以使用 search_knowledge_base 工具从用户上传的知识库文档中检索相关信息来回答问题。"
	}

	systemPrompt := fmt.Sprintf(`你是一个签到系统的智能助手。当前时间是：%s。
你可以帮助用户查询他们的签到记录、考勤统计和积分信息。
当用户询问签到情况时，请使用 get_user_checkin_info 工具获取数据，然后以友好、简洁的方式回答用户。
如果不指定年月，默认查询本月的数据。%s`, now.Format("2006年01月02日"), kbHint)

	req := openai.ChatCompletionRequest{
		Model: s.model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: message},
		},
		Tools: s.buildTools(ctx, userID),
	}

	resp, err := s.client.CreateChatCompletion(ctx, req)
	if err != nil {
		zap.L().Error("CreateChatCompletion error", zap.Error(err))
		return "", fmt.Errorf("AI请求失败: %v", err)
	}

	msg := resp.Choices[0].Message

	// 处理工具调用（支持多个工具）
	if len(msg.ToolCalls) > 0 {
		req.Messages = append(req.Messages, msg)

		for _, toolCall := range msg.ToolCalls {
			result := s.handleToolCall(ctx, userID, toolCall, now)
			req.Messages = append(req.Messages, openai.ChatCompletionMessage{
				Role:       openai.ChatMessageRoleTool,
				Content:    result,
				Name:       toolCall.Function.Name,
				ToolCallID: toolCall.ID,
			})
		}

		resp2, err := s.client.CreateChatCompletion(ctx, req)
		if err != nil {
			zap.L().Error("CreateChatCompletion (2nd) error", zap.Error(err))
			return "", fmt.Errorf("AI请求失败: %v", err)
		}
		return resp2.Choices[0].Message.Content, nil
	}

	return msg.Content, nil
}
