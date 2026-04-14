package model

type MonthDetailOutput struct {
	CheckinDays      []int `json:"checkInDays"`      //签到的日期序号
	RetroCheckinDays []int `json:"retroCheckInDays"` //补签的日期序号
	IsCheckinToday   bool  `json:"isCheckinToday"`   //今天是否签到
	RemainRetroTimes int   `json:"remainRetroTimes"` //剩余补签次数
	ConsectiveDays   int   `json:"consectiveDays"`   //连续签到天数
}
