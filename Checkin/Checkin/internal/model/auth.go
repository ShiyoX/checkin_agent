package model

type LoginInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginOutput struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}
type RefreshTokenOutput struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refresToken"`
}
