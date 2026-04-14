package points

import (
	"Checkin/internal/dao/query"
	"Checkin/internal/model"
	"context"
	"errors"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

func Summary(ctx context.Context, userID int64) (*model.SummaryOutput, error) {
	//1.从数据库查询用户信息
	output := &model.SummaryOutput{}
	upInst, err := query.UserPoint.WithContext(ctx).
		Where(query.UserPoint.UserID.Eq(userID)).
		First()
	if err != nil {
		//排除用户可能没有注册
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return output, nil
		}
		zap.L().Error("查询用户失败", zap.Error(err))
		return nil, err
	}
	output.TotalPoints = upInst.Points
	return output, nil
}

func Records(ctx context.Context, input *model.RecordsInput) (*model.RecordsOutput, error) {
	//1.从数据库分页种查询记录
	var records []*model.UserPointsTransaction
	total, err := query.UserPointsTransaction.WithContext(ctx).
		Where(query.UserPointsTransaction.UserID.Eq(input.UserID)).
		Order(query.UserPointsTransaction.CreatedAt.Desc()).
		Limit(input.Limit).
		ScanByPage(records, input.Offset, input.Limit)
	if err != nil {
		zap.L().Error("查询用户积分记录失败", zap.Error(err))
		return nil, err
	}
	//2.格式化
	list := make([]model.RecordInfo, len(records))
	for _, v := range records {
		list = append(list, model.RecordInfo{
			PointsChange:    v.PointsChange,
			TransactionType: v.TransactionType,
			Description:     v.Description,
			TransactionTime: v.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	hasMore := len(records) == input.Limit && int(total) > input.Offset+input.Limit
	//3.返回数据
	return &model.RecordsOutput{
		Total:   int(total),
		HasMore: hasMore,
		List:    list,
	}, nil
}
