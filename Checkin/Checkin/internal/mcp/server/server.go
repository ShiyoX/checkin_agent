package mcpserver

import (
	"Checkin/internal/dao/query"
	ragSvc "Checkin/internal/service/rag"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type CheckinMCPServer struct {
	mcpServer  *server.MCPServer
	ragService *ragSvc.RAGService
}

func NewCheckinMCPServer(ragService *ragSvc.RAGService) *CheckinMCPServer {
	s := &CheckinMCPServer{
		ragService: ragService,
	}

	s.mcpServer = server.NewMCPServer(
		"checkin-mcp-server",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithLogging(),
	)

	s.registerTools()
	return s
}

func (s *CheckinMCPServer) registerTools() {
	// 工具1：查询签到记录
	s.mcpServer.AddTool(
		mcp.NewTool(
			"get_checkin_info",
			mcp.WithDescription("获取指定用户在某年某月的签到记录和积分信息"),
			mcp.WithNumber("user_id", mcp.Description("用户ID"), mcp.Required()),
			mcp.WithNumber("year", mcp.Description("年份，例如 2026")),
			mcp.WithNumber("month", mcp.Description("月份，1-12")),
		),
		s.handleGetCheckinInfo,
	)

	// 工具2：查询积分
	s.mcpServer.AddTool(
		mcp.NewTool(
			"get_points_summary",
			mcp.WithDescription("获取指定用户的积分摘要，包括当前可用积分和累计获得积分"),
			mcp.WithNumber("user_id", mcp.Description("用户ID"), mcp.Required()),
		),
		s.handleGetPointsSummary,
	)

	// 工具3：知识库检索
	s.mcpServer.AddTool(
		mcp.NewTool(
			"search_knowledge_base",
			mcp.WithDescription("从指定用户上传的知识库文档中检索相关信息"),
			mcp.WithNumber("user_id", mcp.Description("用户ID"), mcp.Required()),
			mcp.WithString("query", mcp.Description("检索查询内容"), mcp.Required()),
		),
		s.handleSearchKnowledgeBase,
	)
}

func (s *CheckinMCPServer) handleGetCheckinInfo(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	userIDFloat, ok := args["user_id"].(float64)
	if !ok {
		return nil, fmt.Errorf("user_id is required")
	}
	userID := int64(userIDFloat)

	now := time.Now()
	year := now.Year()
	month := int(now.Month())

	if y, ok := args["year"].(float64); ok && y > 0 {
		year = int(y)
	}
	if m, ok := args["month"].(float64); ok && m > 0 {
		month = int(m)
	}

	// 查询积分
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
		return nil, fmt.Errorf("获取签到记录失败: %v", err)
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

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{Type: "text", Text: result},
		},
	}, nil
}

func (s *CheckinMCPServer) handleGetPointsSummary(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	userIDFloat, ok := args["user_id"].(float64)
	if !ok {
		return nil, fmt.Errorf("user_id is required")
	}
	userID := int64(userIDFloat)

	userPoint, err := query.UserPoint.WithContext(ctx).
		Where(query.UserPoint.UserID.Eq(userID)).
		First()

	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{Type: "text", Text: "无法获取该用户的积分信息"},
			},
		}, nil
	}

	result := fmt.Sprintf("用户积分摘要:\n当前可用积分: %d\n累计获得积分: %d",
		userPoint.Points, userPoint.PointsTotal)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{Type: "text", Text: result},
		},
	}, nil
}

func (s *CheckinMCPServer) handleSearchKnowledgeBase(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	userIDFloat, ok := args["user_id"].(float64)
	if !ok {
		return nil, fmt.Errorf("user_id is required")
	}
	userID := int64(userIDFloat)

	queryStr, ok := args["query"].(string)
	if !ok || queryStr == "" {
		return nil, fmt.Errorf("query is required")
	}

	if s.ragService == nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{Type: "text", Text: "知识库服务未配置"},
			},
		}, nil
	}

	if !s.ragService.HasUserIndex(ctx, userID) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{Type: "text", Text: "该用户尚未上传知识库文件"},
			},
		}, nil
	}

	docs, err := s.ragService.Retrieve(ctx, userID, queryStr)
	if err != nil {
		return nil, fmt.Errorf("知识库检索失败: %v", err)
	}

	if len(docs) == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{Type: "text", Text: "知识库中未找到相关内容"},
			},
		}, nil
	}

	result := "【知识库检索结果】\n"
	for i, doc := range docs {
		result += fmt.Sprintf("[文档片段 %d]: %s\n\n", i+1, doc)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{Type: "text", Text: result},
		},
	}, nil
}

// Start 启动 MCP HTTP Server
func (s *CheckinMCPServer) Start(addr string) error {
	httpServer := server.NewStreamableHTTPServer(s.mcpServer)
	log.Printf("MCP Server listening on %s/mcp", addr)
	return httpServer.Start(addr)
}
