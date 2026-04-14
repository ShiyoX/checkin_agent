package checkin

import (
	"Checkin/api"
	v1 "Checkin/api/checkin/v1"
	"Checkin/internal/middleware"
	"Checkin/internal/service/checkin"
	"time"

	"github.com/gin-gonic/gin"
)

// 补签接口
func RetroactiveHandler(c *gin.Context) {
	//1. 获取请求参数
	var req v1.RetroReq
	err := c.ShouldBindJSON(&req)
	if err != nil {
		api.ResponseError(c, api.CodeInvalidParam)
	}
	userId := c.Value(middleware.CtxKeyUserID).(int64)
	if userId == 0 {
		api.ResponseError(c, api.CodeNeedLogin)
		return
	}
	//校验日期格式
	t, err := time.Parse(time.DateOnly, req.Date)
	if err != nil {
		api.ResponseErrorWithMsg(c, api.CodeInvalidParam, err.Error())
		return
	}
	//2.调用Service层处理业务
	err = checkin.Retroactive(c, userId, t)
	if err != nil {
		api.ResponseErrorWithMsg(c, api.CodeServerBusy, err.Error())
		return
	}
	//3.返回响应
	api.ResponseSuccess(c, &v1.RetroResp{})
	return

}
