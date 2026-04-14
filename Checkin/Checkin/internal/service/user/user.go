package user

import (
	"Checkin/internal/dao/query"
	"Checkin/internal/model"
	"Checkin/pkg/snowflake"
	"context"
	"errors"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// 业务逻辑层
const (
	defaultAvatar = "https://i0.hdslb.com/bfs/article/d8078267a71b56a214af9b0044108b60b48d7cfb.png@1256w_708h_!web-article-pic.avif"
)

var (
	ErrUserExists = errors.New("用户名已存在")
)

// 创建用户
func Create(ctx context.Context, input *model.CreateUserInput) (*model.CreateUserOutput, error) {
	//1.判断用户名是否存在
	//用Gorm gen查询
	count, err := query.Userinfo.WithContext(ctx).Where(query.Userinfo.Username.Eq(input.Username)).Count()
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, ErrUserExists
	}
	//2.创建用户

	//密码加密
	hashPwd, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	uid, err := snowflake.NextID()
	if err != nil {
		zap.L().Error("Create:generate snowflake ID error!", zap.Error(err))
		return nil, err
	}
	user := &model.Userinfo{
		UserID:   uid,
		Username: input.Username,
		Password: string(hashPwd),
		Email:    input.Email,
		Avatar:   defaultAvatar,
	}

	err = query.Userinfo.WithContext(ctx).Create(user)
	if err != nil {
		zap.L().Error("Create - query.Userinfo.WithContext(ctx).Create(user)", zap.Error(err))
		return nil, err
	}
	return &model.CreateUserOutput{
		UserID:   user.UserID,
		Username: user.Username,
	}, nil
}

func Getprofile(ctx context.Context, userId int64) (*model.UserProfileOutput, error) {
	user, err := query.Userinfo.WithContext(ctx).Where(query.Userinfo.UserID.Eq(userId)).First()
	if err != nil {
		zap.L().Error("Getprofile - query.Userinfo.WithContext(ctx).Where(query.Userinfo.UserID.Eq(userId)).First()", zap.Error(err))
		return nil, err
	}
	return &model.UserProfileOutput{
		UserID:   user.UserID,
		Username: user.Username,
		Email:    user.Email,
		Avatar:   user.Avatar,
	}, nil
}
