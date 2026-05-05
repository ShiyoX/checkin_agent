package mcpclient

import (
	"context"
	"fmt"
	"log"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

type CheckinMCPClient struct {
	c *client.Client
}

// NewCheckinMCPClient 创建 MCP 客户端
func NewCheckinMCPClient(httpURL string) (*CheckinMCPClient, error) {
	httpTransport, err := transport.NewStreamableHTTP(httpURL)
	if err != nil {
		return nil, fmt.Errorf("创建 MCP 传输失败: %w", err)
	}

	c := client.NewClient(httpTransport)
	return &CheckinMCPClient{c: c}, nil
}

// Initialize 初始化客户端，与 Server 握手
func (m *CheckinMCPClient) Initialize(ctx context.Context) (*mcp.InitializeResult, error) {
	m.c.OnNotification(func(notification mcp.JSONRPCNotification) {
		log.Printf("MCP 通知: %s\n", notification.Method)
	})

	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "Checkin MCP Client",
		Version: "1.0.0",
	}
	initRequest.Params.Capabilities = mcp.ClientCapabilities{}

	result, err := m.c.Initialize(ctx, initRequest)
	if err != nil {
		return nil, fmt.Errorf("MCP 初始化失败: %w", err)
	}

	log.Printf("已连接 MCP Server: %s (v%s)",
		result.ServerInfo.Name, result.ServerInfo.Version)
	return result, nil
}

// Ping 健康检查
func (m *CheckinMCPClient) Ping(ctx context.Context) error {
	return m.c.Ping(ctx)
}

// ListTools 列出 Server 上所有可用工具
func (m *CheckinMCPClient) ListTools(ctx context.Context) ([]mcp.Tool, error) {
	result, err := m.c.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return nil, fmt.Errorf("列出工具失败: %w", err)
	}
	return result.Tools, nil
}

// CallTool 调用指定工具
func (m *CheckinMCPClient) CallTool(ctx context.Context, toolName string, args map[string]interface{}) (*mcp.CallToolResult, error) {
	result, err := m.c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      toolName,
			Arguments: args,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("调用工具 %s 失败: %w", toolName, err)
	}
	return result, nil
}

// GetCheckinInfo 查询签到信息的快捷方法
func (m *CheckinMCPClient) GetCheckinInfo(ctx context.Context, userID int64, year, month int) (*mcp.CallToolResult, error) {
	return m.CallTool(ctx, "get_checkin_info", map[string]interface{}{
		"user_id": userID,
		"year":    year,
		"month":   month,
	})
}

// GetPointsSummary 查询积分的快捷方法
func (m *CheckinMCPClient) GetPointsSummary(ctx context.Context, userID int64) (*mcp.CallToolResult, error) {
	return m.CallTool(ctx, "get_points_summary", map[string]interface{}{
		"user_id": userID,
	})
}

// SearchKnowledgeBase 知识库检索的快捷方法
func (m *CheckinMCPClient) SearchKnowledgeBase(ctx context.Context, userID int64, query string) (*mcp.CallToolResult, error) {
	return m.CallTool(ctx, "search_knowledge_base", map[string]interface{}{
		"user_id": userID,
		"query":   query,
	})
}

// GetToolResultText 提取工具返回的文本内容
func (m *CheckinMCPClient) GetToolResultText(result *mcp.CallToolResult) string {
	var text string
	for _, content := range result.Content {
		if tc, ok := content.(mcp.TextContent); ok {
			text += tc.Text + "\n"
		}
	}
	return text
}

// Close 关闭客户端
func (m *CheckinMCPClient) Close() {
	if m.c != nil {
		m.c.Close()
	}
}
