package checkin

import (
	"Checkin/api"
	v1 "Checkin/api/checkin/v1"
	"Checkin/internal/middleware"
	"Checkin/internal/service/checkin"

	"github.com/gin-gonic/gin"
)

// 日常签到接口
func DaylyHandler(c *gin.Context) {
	//1. 获取请求参数userID
	userID := c.Value(middleware.CtxKeyUserID).(int64)
	if userID == 0 {
		api.ResponseError(c, api.CodeNeedLogin)
		return
	}
	//2。调用Service层处理业务
	err := checkin.Daily(c, userID)
	if err != nil {
		api.ResponseErrorWithMsg(c, api.CodeServerBusy, err.Error())
		return
	}
	//3.返回响应
	api.ResponseSuccess(c, v1.DailyResp{})
}
