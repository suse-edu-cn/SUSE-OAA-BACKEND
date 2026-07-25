package service

import (
	"errors"
	"growthos/internal/model"
	"growthos/internal/repository"
	"growthos/internal/request"

	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	Repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) UserService {
	return UserService{
		Repo: repo,
	}
}

func (u *UserService) Register(user model.User) error {
	var err error
	var HashPassword []byte
	HashPassword, err = bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	password := string(HashPassword)
	if err != nil {
		return errors.New("密码加密失败")
	}
	user.Password = password
	return u.Repo.CreateUser(user)
}

func (u *UserService) Login(req request.LoginReq) (model.User, error) {
	user, err := u.Repo.FindUserByAccount(req.Account)
	if err != nil {
		return user, errors.New("获取用户信息失败")
	}
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		return user, errors.New("密码错误")
	}
	return user, nil
}
