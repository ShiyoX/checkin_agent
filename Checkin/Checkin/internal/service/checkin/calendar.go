package checkin

import (
	"Checkin/internal/dao"
	"Checkin/internal/model"
	"context"
	"fmt"
	"go.uber.org/zap"
	"time"
)

func MonthDetail(ctx context.Context, userId int64, t time.Time) (*model.MonthDetailOutput, error) {
	//1.获取当月签到天数
	checkinBitmap, retroBitmap, err := GetMonthBitmap(ctx, userId, t.Year(), int(t.Month()))
	if err != nil {
		zap.L().Error("获取签到记录失败", zap.Error(err))
		return nil, err
	}

	//当月多少天
	firstOfmonth := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.Local)
	lastOfMonth := firstOfmonth.AddDate(0, 1, -1)
	dayNum := lastOfMonth.Day()
	checkinDays := parseBitmapToDays(checkinBitmap, dayNum)
	retroDays := parseBitmapToDays(retroBitmap, dayNum)
	//2.计算连续签到天数
	bitmap := checkinBitmap | retroBitmap
	maxConsective, err := calMonthConsectiveDays(ctx, bitmap, dayNum)
	if err != nil {
		zap.L().Error("计算连续签到天数失败", zap.Error(err))
		return nil, err
	}

	//3.计算剩余补签次数
	remainRetrotimes := MaxRetroTimeMonth - len(retroDays)
	//4.计算当天是否签到
	now := time.Now()
	isCheckedToday := checkinBitmap&(1<<uint(dayNum-now.Day())) != 0
	return &model.MonthDetailOutput{
		CheckinDays:      checkinDays,
		RetroCheckinDays: retroDays,
		ConsectiveDays:   maxConsective,
		RemainRetroTimes: remainRetrotimes,
		IsCheckinToday:   isCheckedToday,
	}, nil

}

func parseBitmapToDays(bitmap uint64, dayNum int) []int {
	days := make([]int, 0, dayNum)
	for i := range dayNum {
		if (bitmap & (1 << uint(dayNum-1-i))) != 0 {
			//第dayNum-i天签到
			days = append(days, i+1)
		}
	}
	return days
}

func IsCheckedToday(ctx context.Context, userId int64) (bool, error) {
	now := time.Now()
	year := now.Year()
	key := fmt.Sprintf(SignKeyFormat, userId, year)
	dayOffset := now.YearDay() - 1
	value, err := dao.RedisClient.GetBit(ctx, key, int64(dayOffset)).Result()
	if err != nil {
		zap.L().Error("获取签到记录失败", zap.Error(err))
		return false, err
	}
	return value == 1, nil
}
