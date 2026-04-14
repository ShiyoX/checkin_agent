package checkin

import (
	"Checkin/api"
	v1 "Checkin/api/checkin/v1"
	"Checkin/internal/middleware"
	"Checkin/internal/service/checkin"
	"time"

	"github.com/gin-gonic/gin"
)

func CalendarHandler(c *gin.Context) {
	//1. 获取请求参数userID
	var req v1.CalendarReq
	err := c.ShouldBindQuery(&req)
	if err != nil {
		api.ResponseError(c, api.CodeInvalidParam)
		return
	}
	userId := c.Value(middleware.CtxKeyUserID).(int64)
	//解析年月
	t, err := time.Parse("2025-01", req.YearMonth)
	if err != nil {
		api.ResponseErrorWithMsg(c, api.CodeInvalidParam, err.Error())
		return
	}
	//2.调用Service层处理业务
	output, err := checkin.MonthDetail(c, userId, t)
	if err != nil {
		api.ResponseErrorWithMsg(c, api.CodeServerBusy, err.Error())
		return
	}
	api.ResponseSuccess(c, &v1.CalendarResp{
		Year:  t.Year(),
		Month: int(t.Month()),
		Detail: v1.DetailInfo{
			CheckinDays:      output.CheckinDays,
			RetroCheckinDays: output.RetroCheckinDays,
			IsCheckinToday:   output.IsCheckinToday,
			RemainRetroTimes: output.RemainRetroTimes,
			ConsectiveDays:   output.ConsectiveDays,
		},
	})
	return
}
