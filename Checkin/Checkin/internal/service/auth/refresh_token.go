package auth

import (
	"Checkin/internal/dao/query"
	"Checkin/internal/model"
	"Checkin/pkg/jwt"
	"context"

	"go.uber.org/zap"
)

func RefreshToken(ctx context.Context, refreshToken string) (*model.RefreshTokenOutput, error) {
	//1.检验RefreshToken
	claims, err := jwt.ParseRefreshToken(refreshToken)
	if err != nil {
		zap.L().Error("RefreshToken校验失败", zap.Error(err))
		return nil, err
	}
	//2.解析得到UserID
	UserID := claims.UserId
	//3.根据UserID查询用户信息
	UserInst, err := query.Userinfo.WithContext(ctx).
		Where(query.Userinfo.UserID.Eq(UserID)).
		First()
	if err != nil {
		zap.L().Error("RefreshToken:查询用户信息失败!", zap.Error(err))
		return nil, err
	}
	//4.生成新的AccessToken和RefreshToken
	accessToken, err := jwt.GenAccessToken(UserInst.UserID, UserInst.Username)
	if err != nil {
		zap.L().Error("RefreshToken:生成AccessToken失败!", zap.Error(err))
		return nil, err
	}
	refreshToken, err = jwt.GenRefreshToken(UserInst.UserID, UserInst.Username)
	if err != nil {
		zap.L().Error("RefreshToken:生成RefreshToken失败!", zap.Error(err))
		return nil, err
	}
	//5.返回新的Token
	return &model.RefreshTokenOutput{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
