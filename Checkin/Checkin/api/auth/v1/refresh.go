package v1

type RefreshReq struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

type RefreshResp struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}
