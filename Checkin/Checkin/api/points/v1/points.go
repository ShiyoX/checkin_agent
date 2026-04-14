package v1

type SummaryResp struct {
	Total int64 `json:"total"`
}

type RecordsRep struct {
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

type RecordsResp struct {
	Total   int          `json:"total"`
	HasMore bool         `json:"hasMore"` //是否还有更多数据
	List    []RecordInfo `json:"list"`
}

type RecordInfo struct {
	PointsChange    int64  `json:"pointsChange"` //积分变动数量
	TransactionType int32  `json:"transactionType"`
	Description     string `json:"description"`
	TransactionTime string `json:"transactionTime"`
}
