package service

import (
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"growthos/internal/model"
	"growthos/internal/repository"
	"growthos/internal/request"
)

type UserService struct {
	Repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) UserService {
	return UserService{Repo: repo}
}

func (u *UserService) Register(req request.RegisterReq) error {
	req.StudentID = strings.TrimSpace(req.StudentID)
	req.Username = strings.TrimSpace(req.Username)
	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(req.Email)
	if req.StudentID == "" || req.Username == "" || req.Name == "" || req.Email == "" || req.Password == "" {
		return errors.New("参数错误")
	}

	if err := u.Repo.CheckExist(req.StudentID, req.Email); err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("密码加密失败")
	}

	user := model.User{
		StudentID: req.StudentID,
		Username:  req.Username,
		Name:      req.Name,
		Email:     req.Email,
		Password:  string(hash),
	}
	if err := u.Repo.CreateUser(user); err != nil {
		return errors.New("创建失败" + err.Error())
	}
	return nil
}

func (u *UserService) Login(req request.LoginReq) (model.User, error) {
	user, err := u.Repo.FindUserByAccount(req.Account)
	if err != nil {
		return user, errors.New("获取用户信息失败")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return user, errors.New("密码错误")
	}
	return user, nil
}
