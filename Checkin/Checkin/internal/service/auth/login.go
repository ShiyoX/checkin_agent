package auth

import (
	"Checkin/internal/dao/query"
	"Checkin/internal/model"
	"Checkin/pkg/jwt"
	"context"
	"errors"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

func Login(ctx context.Context, input *model.LoginInput) (*model.LoginOutput, error) {
	//1.登录校验
	userInst, err := query.Userinfo.WithContext(ctx).
		Where(query.Userinfo.Username.Eq(input.Username)).
		First()
	if err != nil {
		zap.L().Error("Login:查询用户信息失败!", zap.Error(err))
		return nil, errors.New("用户名或密码错误")
	}

	//校验密码
	err = bcrypt.CompareHashAndPassword([]byte(userInst.Password), []byte(input.Password))
	if err != nil {
		zap.L().Error("Login:密码校验失败!", zap.Error(err))
		return nil, errors.New("用户名或密码错误")
	}
	//2.如果登陆成功,生成Token
	//2.1 生成AccessToken
	accessToken, err := jwt.GenAccessToken(userInst.UserID, userInst.Username)
	if err != nil {
		zap.L().Error("Login:生成AccessToken失败!", zap.Error(err))
		return nil, errors.New("服务器繁忙，请稍后再试")
	}
	//2.2 生成RefreshToken
	refreshToken, err := jwt.GenRefreshToken(userInst.UserID, userInst.Username)
	if err != nil {
		zap.L().Error("Login:生成RefreshToken失败!", zap.Error(err))
		return nil, errors.New("服务器繁忙，请稍后再试")
	}
	//3.返回Token
	return &model.LoginOutput{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, err
}
