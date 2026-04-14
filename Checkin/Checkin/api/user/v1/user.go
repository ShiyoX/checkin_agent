package v1

// 创建请求
type CreateReq struct {
	Username        string `json:"username" binding:"required"`
	Password        string `json:"password" binding:"required"`
	Email           string `json:"email" binding:"required,contains=@"`
	ComfirmPassword string `json:"comfirmPassword" binding:"required,eqfield=Password"`
}

type CreateResp struct {
	UserID   int64  `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

type MeResp struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Avatar   string `json:"avator"`
}
