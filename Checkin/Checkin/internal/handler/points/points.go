package points

import (
	"Checkin/api"
	v1 "Checkin/api/points/v1"
	"Checkin/internal/middleware"
	"Checkin/internal/model"
	"Checkin/internal/service/points"

	"github.com/gin-gonic/gin"
)

const (
	defaultLimit  = 10
	defaultOffset = 0
	maxLimit      = 50
)

func SummaryHandler(c *gin.Context) {
	//1. 获取请求参数
	userID := c.Value(middleware.CtxKeyUserID).(int64)
	if userID == 0 {
		api.ResponseError(c, api.CodeNeedLogin)
		return
	}

	//2.调用Service层处理业务
	output, err := points.Summary(c, userID)
	if err != nil {
		api.ResponseErrorWithMsg(c, api.CodeServerBusy, err.Error())
		return
	}
	//3.返回响应
	api.ResponseSuccess(c, &v1.SummaryResp{
		Total: output.TotalPoints,
	})
	return
}

func RecordsHandler(c *gin.Context) {
	//1.获取当前用户信息和分页信息
	var req v1.RecordsRep
	err := c.ShouldBind(&req)
	if err != nil {
		api.ResponseError(c, api.CodeInvalidParam)
		return
	}
	userID := c.Value(middleware.CtxKeyUserID).(int64)
	if userID == 0 {
		api.ResponseError(c, api.CodeNeedLogin)
		return
	}
	//分页参数校验
	if req.Limit <= 0 || req.Limit > maxLimit {
		req.Limit = defaultLimit
	}
	if req.Offset < 0 {
		req.Offset = defaultOffset
	}
	//2.调用Service层处理业务
	output, err := points.Records(c, &model.RecordsInput{
		UserID: userID,
		Limit:  req.Limit,
		Offset: req.Offset,
	})
	if err != nil {
		api.ResponseErrorWithMsg(c, api.CodeServerBusy, err.Error())
		return
	}
	list := make([]v1.RecordInfo, len(output.List))
	for i, v := range output.List {
		list[i] = v1.RecordInfo{
			PointsChange:    v.PointsChange,
			TransactionType: v.TransactionType,
			Description:     v.Description,
			TransactionTime: v.TransactionTime,
		}
	}
	//3.返回响应
	api.ResponseSuccess(c, &v1.RecordsResp{
		Total:   output.Total,
		HasMore: output.HasMore,
		List:    list,
	})
	return
}
