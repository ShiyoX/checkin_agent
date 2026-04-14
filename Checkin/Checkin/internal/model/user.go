package model

type CreateUserInput struct {
	Username        string `json:"username"`
	Password        string `json:"password"`
	Email           string `json:"email" `
	ComfirmPassword string `json:"comfirm_password"`
}

type CreateUserOutput struct {
	UserID   int64  `json:"id"`
	Username string `json:"username"`
}

type UserProfileOutput struct {
	UserID   int64  `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Avatar   string `json:"avatar"`
}
