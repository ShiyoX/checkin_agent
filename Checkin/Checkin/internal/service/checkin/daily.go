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

var (
	ErrAlreadyCheckedIn = errors.New("already checked in")
)

const (
	SignKeyFormat       = "user:checkin:daily:%d:%d" // 签到记录的Redis Key格式，%d为用户ID,和年份
	MonthRetroKeyFormat = "user:checkins:retro:%d:%d:%02d"
)

const (
	//默认积分
	DefaultPoints = 1
	//最大补签次数
	MaxRetroTimeMonth = 3
)

// 积分变更的type
type PointsTransactionType int32

const (
	PointsTransactionTypeDaily       PointsTransactionType = 1 // 签到积分
	PointsTransactionTypeConsective  PointsTransactionType = 2 // 连续签到奖励积分
	PointsTransactionTypeRetroactive PointsTransactionType = 3 // 补签积分
)

var PointsTransactionTypeMap = map[PointsTransactionType]string{
	PointsTransactionTypeDaily:       "每日签到奖励",
	PointsTransactionTypeConsective:  "连续签到奖励",
	PointsTransactionTypeRetroactive: "补签%s消耗",
}

type ConsectiveBonusType int32

const (
	ConsectiveBonusType3  ConsectiveBonusType = 1
	ConsectiveBonusType7  ConsectiveBonusType = 2
	ConsectiveBonusType15 ConsectiveBonusType = 3
	ConsectiveBonusType30 ConsectiveBonusType = 4
)

var ConsectiveBonusName = map[ConsectiveBonusType]string{
	ConsectiveBonusType3:  "连续签到3天奖励",
	ConsectiveBonusType7:  "连续签到7天奖励",
	ConsectiveBonusType15: "连续签到15天奖励",
	ConsectiveBonusType30: "连续签到30天奖励",
}

// 连续签到触发规则
type ConsectiveBonusRule struct {
	TriggerDays int                 // 连续签到天数
	Points      int                 // 奖励积分
	Type        ConsectiveBonusType // 奖励类型
}

var ConsectiveBonusRuleList = []ConsectiveBonusRule{
	{TriggerDays: 3, Points: 5, Type: ConsectiveBonusType3},
	{TriggerDays: 7, Points: 10, Type: ConsectiveBonusType7},
	{TriggerDays: 15, Points: 20, Type: ConsectiveBonusType15},
	{TriggerDays: 28, Points: 100, Type: ConsectiveBonusType30},
}

// 每日签到
func Daily(ctx context.Context, userID int64) error {
	//1.获取今天是哪一天，算出偏移量offset
	now := time.Now()
	year := now.Year()
	key := fmt.Sprintf(SignKeyFormat, userID, year)
	offset := now.YearDay() - 1 // 计算偏移量，今天是今年的第几天，减去1得到从0开始的偏移量
	//2.执行bitset操作
	ret, err := dao.RedisClient.SetBit(ctx, key, int64(offset), 1).Result()
	if err != nil {
		zap.L().Error("SetBit Error", zap.Error(err))
		return err
	}
	zap.L().Sugar().Debugf("----> userID: %d, year: %d, offset: %d, ret: %d", userID, year, offset, ret)
	if ret == int64(1) {
		// 如果之前已经签到过了，就不再发放积分了
		return ErrAlreadyCheckedIn
	}
	//3.发放每日签到积分
	err = addPoint(ctx, &model.AddPointInput{
		UserID:     userID,
		PointAmout: DefaultPoints,
		Type:       int32(PointsTransactionTypeDaily),
	})
	if err != nil {
		zap.L().Error("发放签到积分失败", zap.Error(err))
		return err
	}
	//在事务中更新UserPoint表
	//4.发放连续签到奖励积分
	return updateConsectiveBonus(ctx, userID, year, int(now.Month()))
}

// 更新连续签到奖励
func updateConsectiveBonus(ctx context.Context, userID int64, year int, month int) error {
	//1.获取当前的连续签到天数
	checkinBitmap, retroBitmap, err := GetMonthBitmap(ctx, userID, year, month)
	if err != nil {
		zap.L().Error("获取签到记录失败", zap.Error(err))
		return err
	}
	firstOfmonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	lastOfMonth := firstOfmonth.AddDate(0, 1, -1)
	dayNum := lastOfMonth.Day()
	bitmap := retroBitmap | checkinBitmap
	maxConsective, err := calMonthConsectiveDays(ctx, bitmap, dayNum)
	if err != nil {
		zap.L().Error("计算连续签到天数失败", zap.Error(err))
		return err
	}
	//2.获取连续奖励的签到积分
	//2.1查询当月连续签到奖励
	bonusLogList, err := query.UserMonthlyBonusLog.WithContext(ctx).
		Where(query.UserMonthlyBonusLog.UserID.Eq(userID)).
		Where(query.UserMonthlyBonusLog.YearMonth.Eq(fmt.Sprintf("%0d%02d", year, month))).
		Find()
	if err != nil {
		zap.L().Error("查询连续签到奖励失败", zap.Error(err))
		return err
	}
	bonusLogMap := make(map[ConsectiveBonusType]bool, len(bonusLogList))
	for _, log := range bonusLogList {
		bonusLogMap[ConsectiveBonusType(log.BonusType)] = true
	}

	for _, rule := range ConsectiveBonusRuleList {
		if maxConsective >= rule.TriggerDays && !bonusLogMap[rule.Type] {
			//满足条件，发放奖励
			err := addPoint(ctx, &model.AddPointInput{
				UserID:     userID,
				PointAmout: int64(rule.Points),
				Type:       int32(PointsTransactionTypeConsective),
				Desc:       ConsectiveBonusName[rule.Type],
			})
			if err != nil {
				zap.L().Error("发放连续签到奖励失败", zap.Error(err))
				return err
			}
			err = query.UserMonthlyBonusLog.WithContext(ctx).Create(&model.UserMonthlyBonusLog{
				UserID:      userID,
				YearMonth:   fmt.Sprintf("%0d%02d", year, month),
				BonusType:   int32(rule.Type),
				Description: ConsectiveBonusName[rule.Type],
			})
			if err != nil {
				zap.L().Error("记录连续签到奖励失败", zap.Error(err))
				continue
			}
			//更新user_points表，增加积分
			//更新user_points_transaction表，记录积分变动

		}
	}
	//3.更新user_points和user_points_transaction表，记录连续签到奖励
	return nil
}

func GetMonthBitmap(ctx context.Context, userID int64, year, month int) (uint64, uint64, error) {
	//1.获取月度正常签到数据
	firstOfmonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	lastOfMonth := firstOfmonth.AddDate(0, 1, -1)
	dayNum := lastOfMonth.Day()
	offset := firstOfmonth.Day() - 1
	bitWidthType := fmt.Sprintf("u%d", dayNum)
	key := fmt.Sprintf(SignKeyFormat, year, userID)
	value, err := dao.RedisClient.BitField(ctx, key, "GET", bitWidthType, offset).Result()
	if err != nil {
		zap.L().Error("BitField Error", zap.Error(err))
		return 0, 0, err
	}
	if len(value) == 0 {
		value = []int64{0}
	}
	checkinBitmap := uint64(value[0])
	//2.获取月度补签数据
	retroKey := fmt.Sprintf(MonthRetroKeyFormat, userID, year, month)
	retroValues, err := dao.RedisClient.BitField(ctx, retroKey, "GET", bitWidthType, "#0").Result()
	if err != nil {
		zap.L().Error("获取补签记录失败", zap.Error(err))
		return 0, 0, err
	}
	if len(retroValues) == 0 {
		retroValues = []int64{0}
	}
	retroBitmap := uint64(retroValues[0])
	return checkinBitmap, retroBitmap, nil
}

func calMonthConsectiveDays(ctx context.Context, bitmap uint64, dayNum int) (int, error) {

	//3.计算连续签到天数
	//bitmap := retroBitmap | checkinBitmap
	maxCount := 0
	curCount := 0
	for i := range dayNum {
		if (bitmap & (1 << uint(dayNum-1-i))) != 0 {
			curCount++
			if curCount > maxCount {
				maxCount = curCount
			}
		} else {
			curCount = 0
		}
	}
	if curCount > maxCount {
		maxCount = curCount
	}
	return maxCount, nil
}

func addPoint(ctx context.Context, input *model.AddPointInput) error {
	//更新user_points表，增加积分
	//3.1.查询user_points表
	//如果没有记录，插入一条记录 ,如果有记录，更新积分字段
	userPoint, err := query.UserPoint.WithContext(ctx).
		Where(query.UserPoint.UserID.Eq(input.UserID)).
		First()
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		zap.L().Error("Query Error", zap.Error(err))
		return err
	}
	if userPoint == nil || input.UserID == 0 {
		//没有记录，插入一条记录
		userPoint = &model.UserPoint{
			UserID: input.UserID,
		}
	}
	userPoint.Points = userPoint.Points + input.PointAmout
	userPoint.PointsTotal = userPoint.PointsTotal + input.PointAmout

	err = query.Q.Transaction(func(tx *query.Query) error {
		if err := tx.UserPoint.WithContext(ctx).Save(userPoint); err != nil {
			zap.L().Error("Save Error", zap.Error(err))
			return err
		}
		//更新user_point_transaction表，记录积分变动
		if err := tx.UserPointsTransaction.WithContext(ctx).Create(&model.UserPointsTransaction{
			UserID:          input.UserID,
			PointsChange:    input.PointAmout,
			CurrentBalance:  userPoint.Points,
			TransactionType: input.Type,
			Description:     PointsTransactionTypeMap[PointsTransactionTypeDaily],
			CreatedAt:       time.Now(),
		}); err != nil {
			zap.L().Error("Create Error", zap.Error(err))
			return err
		}
		return nil
	})
	if err != nil {
		zap.L().Error("Transaction Error", zap.Error(err))
	}
	return nil
}
