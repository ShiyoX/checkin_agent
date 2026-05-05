package rag

import (
	"Checkin/api"
	"Checkin/internal/middleware"
	ragSvc "Checkin/internal/service/rag"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var ragService *ragSvc.RAGService

func InitRAGService(cfg *viper.Viper) {
	ragService = ragSvc.NewRAGService(cfg)
}

func GetRAGService() *ragSvc.RAGService {
	return ragService
}

type UploadResponse struct {
	FilePath string `json:"file_path"`
	Message  string `json:"message"`
}

// UploadHandler 处理知识库文件上传
func UploadHandler(c *gin.Context) {
	userID, exists := c.Get(middleware.CtxKeyUserID)
	if !exists {
		api.ResponseError(c, api.CodeNeedLogin)
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		api.ResponseErrorWithMsg(c, api.CodeInvalidParam, "请上传文件")
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		api.ResponseErrorWithMsg(c, api.CodeServerBusy, "读取文件失败")
		return
	}

	filePath, err := ragService.UploadAndIndex(
		c.Request.Context(),
		userID.(int64),
		header.Filename,
		content,
	)
	if err != nil {
		zap.L().Error("RAG upload failed", zap.Error(err))
		api.ResponseErrorWithMsg(c, api.CodeServerBusy, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "知识库文件上传并索引成功",
		"data": UploadResponse{
			FilePath: filePath,
			Message:  "文件已成功向量化并存入知识库",
		},
	})
}

// StatusHandler 查询用户知识库状态
func StatusHandler(c *gin.Context) {
	userID, exists := c.Get(middleware.CtxKeyUserID)
	if !exists {
		api.ResponseError(c, api.CodeNeedLogin)
		return
	}

	hasIndex := ragService.HasUserIndex(c.Request.Context(), userID.(int64))
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"has_knowledge_base": hasIndex,
		},
	})
}
