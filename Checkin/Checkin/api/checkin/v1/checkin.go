package v1

type DailyResp struct {
}

type CalendarReq struct {
	YearMonth string `form:"yearMonth" binding:"required"` //格式为 "2025-06"
}

type CalendarResp struct {
	Year   int        `json:"year"`
	Month  int        `json:"month"`
	Detail DetailInfo `json:"detail"`
}

type DetailInfo struct {
	CheckinDays      []int `json:"checkInDays"`      //签到的日期序号
	RetroCheckinDays []int `json:"retroCheckInDays"` //补签的日期序号
	IsCheckinToday   bool  `json:"isCheckinToday"`   //今天是否签到
	RemainRetroTimes int   `json:"remainRetroTimes"` //剩余补签次数
	ConsectiveDays   int   `json:"consectiveDays"`   //连续签到天数
}

type RetroReq struct {
	Date string `form:"date" binding:"required"`
}

type RetroResp struct {
}
