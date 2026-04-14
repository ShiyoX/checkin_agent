package model

type CreateUserInput struct {
	Username        string `json:"username"`
	Password        string `json:"password"`
	Email           string `json:"email"`
	ComfirmPassword string `json:"comfirmPassword"`
}
