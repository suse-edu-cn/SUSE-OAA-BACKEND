package service

import (
	"context"
	"errors"
	"growthos/pkg/utils"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"growthos/internal/model"
	"growthos/internal/repository"
	"growthos/internal/request"
)

type UserService struct {
	Repo           repository.UserRepository
	RoleRepo       repository.RoleRepository
	DepartmentRepo repository.DepartmentRepository
}

func NewUserService(repo repository.UserRepository, roleRepo repository.RoleRepository, departmentRepo repository.DepartmentRepository) UserService {
	return UserService{
		Repo:           repo,
		RoleRepo:       roleRepo,
		DepartmentRepo: departmentRepo,
	}
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
	role, err := u.RoleRepo.FindByName("会员")
	user := model.User{
		StudentID: req.StudentID,
		Username:  req.Username,
		Name:      req.Name,
		Email:     req.Email,
		Password:  string(hash),
		Role:      role,
		RoleID:    &role.ID,
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

func (u *UserService) GetUserInfo(id uint64) (model.UserInfo, error) {
	info, err := u.Repo.GetUserInfoById(id)
	if err != nil {
		return model.UserInfo{}, errors.New("查询失败" + err.Error())
	}
	return info, nil
}

func (u *UserService) SaveRefreshToken(id uint64, device string, time uint) (string, error) {
	token, err := utils.GetUUID()
	if err != nil {
		return "", errors.New("生成refresh_token失败")
	}
	err = u.Repo.SaveRefreshToken(id, device, token, time, context.Background())
	if err != nil {
		return "", err
	}
	return token, nil
}
func (u *UserService) DeleteRefreshToken(id uint64, device string) error {
	err := u.Repo.DeleteRefreshToken(id, device, context.Background())
	if err != nil {
		return err
	}
	return nil
}
func (u *UserService) GetRefreshToken(id uint64, device string) (string, error) {
	token, err := u.Repo.GetRefreshToken(id, device, context.Background())
	if err != nil {
		return "", err
	}
	return token, nil
}
