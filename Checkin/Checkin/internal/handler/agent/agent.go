package agent

import (
	"Checkin/api"
	"Checkin/internal/middleware"
	agentSvc "Checkin/internal/service/agent"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var agentService *agentSvc.AgentService

func InitAgentService(cfg *viper.Viper) {
	agentService = agentSvc.NewAgentService(cfg)
}

type ChatRequest struct {
	Message string `json:"message" binding:"required"`
}

type ChatResponse struct {
	Reply string `json:"reply"`
}

func ChatHandler(c *gin.Context) {
	userID, exists := c.Get(middleware.CtxKeyUserID)
	if !exists {
		api.ResponseError(c, api.CodeNeedLogin)
		return
	}

	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.ResponseError(c, api.CodeInvalidParam)
		return
	}

	// 调用服务处理对话
	reply, err := agentService.Chat(c.Request.Context(), userID.(int64), req.Message)
	if err != nil {
		zap.L().Error("Agent chat failed", zap.Error(err))
		api.ResponseErrorWithMsg(c, api.CodeServerBusy, "AI助手暂时不可用")
		return
	}

	api.ResponseSuccess(c, ChatResponse{
		Reply: reply,
	})
}
