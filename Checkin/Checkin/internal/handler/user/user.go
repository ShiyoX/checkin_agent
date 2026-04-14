package user

import (
	"Checkin/api"
	"Checkin/internal/middleware"
	"Checkin/internal/model"
	"Checkin/internal/service/user"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	v1 "Checkin/api/user/v1"
)

//handler

func CreateHandler(c *gin.Context) {
	//1.获取请求参数和校验参数
	var req v1.CreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		//请求有问题
		zap.L().Error("CreateHandler - ShouldBindJSON", zap.Error(err))
		api.ResponseError(c, api.CodeInvalidParam)
		return
	}

	zap.L().Sugar().Debugf("CreateHandler - ShouldBindJSON", zap.Any("req", req))
	// 2. 执行业务逻辑
	input := &model.CreateUserInput{
		Username: req.Username,
		Password: req.Password,
		Email:    req.Email,
	}
	output, err := user.Create(c, input)
	if err != nil {
		api.ResponseError(c, api.CodeServerBusy)
		return
	}
	api.ResponseSuccess(c, output)
}

func ProfileHandler(c *gin.Context) {
	//1.获取请求参数和校验码
	//从请求头种获取token，解析token，获取用户ID
	//tokenStr := c.GetHeader("Authorization")
	//claims, err := jwt.ParseAccessToken(tokenStr)
	//if err != nil {
	//	zap.L().Error("ProfileHandler - ParseToken", zap.Error(err))
	//	api.ResponseError(c, api.CodeInvalidToken)
	//	return
	//}
	userID := c.Value(middleware.CtxKeyUserID).(int64)
	if userID == 0 {
		zap.L().Error("ProfileHandler - GetUserID", zap.Any("userID", userID))
		api.ResponseError(c, api.CodeInvalidToken)
		return
	}
	zap.L().Sugar().Debugf("ProfileHandler - ParseToken", zap.Any("userID", userID))
	//2.执行业务逻辑
	output, err := user.Getprofile(c, userID)
	if err != nil {
		zap.L().Error("ProfileHandler - Getprofile", zap.Error(err))
		api.ResponseError(c, api.CodeServerBusy)
		return
	}
	//3.返回响应
	api.ResponseSuccess(c, &v1.MeResp{
		Avatar:   output.Avatar,
		Username: output.Username,
		Email:    output.Email,
	})
}
