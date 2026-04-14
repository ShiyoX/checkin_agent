package auth

import (
	"Checkin/api"
	v1 "Checkin/api/auth/v1"
	"Checkin/internal/model"
	"Checkin/internal/service/auth"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func LoginHandler(c *gin.Context) {
	//1.获取请求参数和校验参数
	var req v1.LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		//请求有问题
		zap.L().Error("参数校验失败", zap.Error(err))
		api.ResponseError(c, api.CodeInvalidParam)
		return
	}
	//2.调用用户登录服务
	output, err := auth.Login(c, &model.LoginInput{Username: req.Username, Password: req.Password})
	if err != nil {
		zap.L().Error("用户登录失败", zap.Error(err))
		api.ResponseError(c, api.CodeInvalidToken)
		return
	}
	//3.返回响应
	api.ResponseSuccess(c, v1.LoginResp{
		AccessToken:  output.AccessToken,
		RefreshToken: output.RefreshToken,
	})
}

func RefreshHandler(c *gin.Context) {
	//1.获取请求参数和校验参数
	var req v1.RefreshReq
	if err := c.ShouldBindJSON(&req); err != nil {
		//请求有问题
		zap.L().Error("参数校验失败", zap.Error(err))
		api.ResponseError(c, api.CodeInvalidParam)
		return
	}
	//2.调用Service层刷新Token
	output, err := auth.RefreshToken(c, req.RefreshToken)
	if err != nil {
		zap.L().Error("刷新Token失败", zap.Error(err))
		api.ResponseError(c, api.CodeInvalidToken)
		return
	}
	//3.返回响应
	api.ResponseSuccess(c, v1.RefreshResp{
		AccessToken:  output.AccessToken,
		RefreshToken: output.RefreshToken,
	})
}
