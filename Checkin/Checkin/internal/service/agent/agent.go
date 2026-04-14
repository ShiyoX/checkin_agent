package agent

import (
	"Checkin/internal/dao/query"
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
	client *openai.Client
	model  string
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
		client: openai.NewClientWithConfig(config),
		model:  model,
	}
}

// getUserCheckinInfo 查询用户指定月份的签到信息和总积分
func (s *AgentService) getUserCheckinInfo(ctx context.Context, userID int64, year int, month int) (string, error) {
	// 获取积分信息
	userPoint, err := query.UserPoint.WithContext(ctx).
		Where(query.UserPoint.UserID.Eq(userID)).
		First()
	
	pointsInfo := "无法获取积分信息"
	if err == nil && userPoint != nil {
		pointsInfo = fmt.Sprintf("当前可用积分: %d, 累计获得积分: %d", userPoint.Points, userPoint.PointsTotal)
	}

	// 查询签到记录
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
		daysMap[day] = record.CheckinType // 1=正常签到，2=补签
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

type checkinArgs struct {
	Year  int `json:"year"`
	Month int `json:"month"`
}

// Chat 处理用户对话
func (s *AgentService) Chat(ctx context.Context, userID int64, message string) (string, error) {
	if s.client == nil || s.model == "" {
		return "系统未正确配置AI助手，请联系管理员。", nil
	}

	now := time.Now()
	systemPrompt := fmt.Sprintf(`你是一个签到系统的智能助手。当前时间是：%s。
你可以帮助用户查询他们的签到记录、考勤统计和积分信息。
当用户询问签到情况时，请使用 get_user_checkin_info 工具获取数据，然后以友好、简洁的方式回答用户。
如果不指定年月，默认查询本月的数据。`, now.Format("2006年01月02日"))

	req := openai.ChatCompletionRequest{
		Model: s.model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: systemPrompt,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: message,
			},
		},
		Tools: []openai.Tool{
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
		},
	}

	resp, err := s.client.CreateChatCompletion(ctx, req)
	if err != nil {
		zap.L().Error("CreateChatCompletion error", zap.Error(err))
		return "", fmt.Errorf("AI请求失败: %v", err)
	}

	msg := resp.Choices[0].Message

	// 检查模型是否决定调用工具
	if len(msg.ToolCalls) > 0 {
		toolCall := msg.ToolCalls[0]
		if toolCall.Function.Name == "get_user_checkin_info" {
			var args checkinArgs
			if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
				zap.L().Error("解析工具参数失败", zap.Error(err))
				return "", fmt.Errorf("解析参数失败")
			}
			
			// 补充默认年月（以防模型抽风没传）
			if args.Year == 0 {
				args.Year = now.Year()
			}
			if args.Month == 0 {
				args.Month = int(now.Month())
			}

			// 调用本地工具获取数据
			dataResult, err := s.getUserCheckinInfo(ctx, userID, args.Year, args.Month)
			if err != nil {
				dataResult = fmt.Sprintf("获取数据失败: %v", err)
			}

			// 将工具执行结果返回给大模型继续生成
			req.Messages = append(req.Messages, msg)
			req.Messages = append(req.Messages, openai.ChatCompletionMessage{
				Role:       openai.ChatMessageRoleTool,
				Content:    dataResult,
				Name:       toolCall.Function.Name,
				ToolCallID: toolCall.ID,
			})

			// 第二次调用大模型
			resp2, err := s.client.CreateChatCompletion(ctx, req)
			if err != nil {
				zap.L().Error("CreateChatCompletion (2nd) error", zap.Error(err))
				return "", fmt.Errorf("AI请求失败: %v", err)
			}
			return resp2.Choices[0].Message.Content, nil
		}
	}

	// 如果没有调用工具，直接返回文本
	return msg.Content, nil
}
