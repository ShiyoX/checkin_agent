package main

import (
	"flag"
	"fmt"

	"Checkin/internal/conf"
	"Checkin/internal/dao"
	ragHandler "Checkin/internal/handler/rag"
	mcpserver "Checkin/internal/mcp/server"
	"Checkin/internal/server"
	"Checkin/pkg/jwt"
	"Checkin/pkg/logging"
	"Checkin/pkg/snowflake"
	"go.uber.org/zap"
)

var confPath = flag.String("conf", "./config/config.yaml", "配置文件路径")

func main() {
	// 加载配置
	flag.Parse()
	cfg := conf.Load(*confPath)

	// 初始化日志
	logger, err := logging.NewLogger(cfg)
	if err != nil {
		fmt.Printf("init logger failed, err:%v\n", err)
		return
	}
	defer logger.Sync()

	dao.MustInitMySQL(cfg)  // 初始化 MySQL 连接
	dao.MustInitRedis(cfg)  // 初始化 Redis
	jwt.MustInit(cfg)       // 初始化 jwt
	snowflake.MustInit(cfg) // 初始化 snowflake

	// 初始化路由
	r := server.SetupRoutes(cfg)

	// 启动 MCP Server（独立 goroutine）
	if cfg.GetBool("mcp.enabled") {
		mcpPort := cfg.GetInt("mcp.port")
		mcpSrv := mcpserver.NewCheckinMCPServer(ragHandler.GetRAGService())
		go func() {
			if err := mcpSrv.Start(fmt.Sprintf(":%d", mcpPort)); err != nil {
				zap.L().Error("MCP Server 启动失败", zap.Error(err))
			}
		}()
		zap.L().Info("MCP Server 已启动", zap.Int("port", mcpPort))
	}

	// 启动 HTTP 服务
	err = r.Run(fmt.Sprintf(":%d", cfg.GetInt("server.port")))
	if err != nil {
		panic(err)
	}
}
