package model

type AddPointInput struct {
	UserID     int64  `json:"user_id"`
	PointAmout int64  `json:"points"`
	Type       int32  `json:"type"` // "checkin" 或 "other"
	Desc       string `json:"desc"`
}

type SummaryOutput struct {
	TotalPoints int64 `json:"totalPoints"`
}

type RecordsInput struct {
	UserID int64 `json:"userId"`
	Offset int   `json:"offset"`
	Limit  int   `json:"limit"`
}

type RecordInfo struct {
	PointsChange    int64  `json:"pointsChange"` //积分变动数量
	TransactionType int32  `json:"transactionType"`
	Description     string `json:"description"`
	TransactionTime string `json:"transactionTime"`
}

type RecordsOutput struct {
	Total   int          `json:"total"`
	HasMore bool         `json:"hasMore"` //是否还有更多数据
	List    []RecordInfo `json:"list"`
}
