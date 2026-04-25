package checkin

import (
	"Checkin/internal/dao"
	"Checkin/internal/dao/query"
	"Checkin/internal/model"
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	defaultRetroPointCost = 1 //默认补签积分
)

func Retroactive(ctx context.Context, userId int64, date time.Time) error {
	//1.校验补签日期(业务逻辑version)
	err := checkRetroactiveDate(ctx, userId, date)
	if err != nil {
		zap.L().Error("checkRetroactiveDate Error", zap.Error(err))
		return err
	}
	//2.执行补签
	//在Redis中标记与Setbit
	key := fmt.Sprintf(MonthRetroKeyFormat, userId, date.Year(), int(date.Month()))
	offset := date.Day() - 1
	err = dao.RedisClient.SetBit(ctx, key, int64(offset), 1).Err()
	if err != nil {
		zap.L().Error("Redis SetBit Error", zap.Error(err))
		return err
	}
	//增加积分记录到数据库
	//如果补签失败需要回滚
	if err := RetroWithTransaction(ctx, userId, date); err != nil {
		rollbackErr := dao.RedisClient.SetBit(ctx, key, int64(offset), 0).Err()
		if rollbackErr != nil {
			zap.L().Error("Redis Rollback SetBit Error", zap.Error(rollbackErr))
			return fmt.Errorf("Redis Rollback SetBit Error: %w (original: %v)", rollbackErr, err)
		}
		return err
	}

	//发放可能存在的连续签到奖励（如果补签后连续签到天数达到某个阈值）
	return updateConsectiveBonus(ctx, userId, date.Year(), int(date.Month()))
}

var (
	ErrDateInvalid    = errors.New("补签日期无效")
	ErrNoRetroTimes   = errors.New("本月补签次数已用完")
	ErrNotEnoughPoint = errors.New("积分不足，无法补签")
)

func checkRetroactiveDate(ctx context.Context, userId int64, date time.Time) error {
	//补签日期不能是今天或未来（按“日期”判断，而不是精确到时分秒）
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	checkDate := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	if !checkDate.Before(today) {
		return ErrDateInvalid
	}
	//且只能补签当月的日期
	if checkDate.Year() != today.Year() || checkDate.Month() != today.Month() {
		return ErrDateInvalid
	}
	//不能补签已经签到的日期
	checkinBitmap, retroBitmap, err := GetMonthBitmap(ctx, userId, date.Year(), int(date.Month()))
	if err != nil {
		zap.L().Error("GetMonthBitmap Error", zap.Error(err))
		return ErrDateInvalid
	}
	bitmap := checkinBitmap | retroBitmap
	//当月天数（用于计算“某天对应的bit位置”）
	monthDays := time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, date.Location()).AddDate(0, 1, -1).Day()
	if bitmap&(1<<uint(monthDays-date.Day())) != 0 {
		return ErrDateInvalid
	}
	//补签次数校验：每月补签次数不超过 MaxRetroTimeMonth
	count := 0
	for i := 0; i < monthDays; i++ {
		if (retroBitmap & (1 << uint(monthDays-1-i))) != 0 {
			count++
		}
	}
	if count >= MaxRetroTimeMonth {
		return ErrNoRetroTimes
	}
	return nil
}

func RetroWithTransaction(ctx context.Context, userId int64, date time.Time) error {
	return query.Q.Transaction(func(tx *query.Query) error {
		//1.查询当前用户积分
		var (
			upInst *model.UserPoint
			err    error
		)
		upInst, err = tx.UserPoint.WithContext(ctx).
			Where(tx.UserPoint.UserID.Eq(userId)).
			First()
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			upInst = &model.UserPoint{
				UserID: userId,
			}
		}
		if upInst.Points < defaultRetroPointCost {
			return ErrNotEnoughPoint
		}
		//2.扣除积分
		pointsChange := -defaultRetroPointCost + DefaultPoints
		newPoints := upInst.Points + int64(pointsChange)

		//3.积分记录
		retroCostTransaction := &model.UserPointsTransaction{
			UserID:          userId,
			PointsChange:    defaultRetroPointCost,
			CurrentBalance:  newPoints,
			TransactionType: int32(PointsTransactionTypeRetroactive),
			Description:     fmt.Sprintf(PointsTransactionTypeMap[PointsTransactionTypeRetroactive], date.Format(time.DateOnly)),
		}
		if err := tx.UserPointsTransaction.WithContext(ctx).Create(retroCostTransaction); err != nil {
			zap.L().Error("Create UserPointsTransaction Error", zap.Error(err))
			return err
		}
		upInst.Points = newPoints //当前积分值
		if err := tx.UserPoint.WithContext(ctx).Save(upInst); err != nil {
			zap.L().Error("Save UserPoint Error", zap.Error(err))
			return err
		}
		return nil
	})

}
