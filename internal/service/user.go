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
	Email          EmailService
}

func NewUserService(repo repository.UserRepository, roleRepo repository.RoleRepository, departmentRepo repository.DepartmentRepository, email EmailService) UserService {
	return UserService{
		Repo:           repo,
		RoleRepo:       roleRepo,
		DepartmentRepo: departmentRepo,
		Email:          email,
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
	}
	if err := u.Repo.CreateUser(user); err != nil {
		return errors.New("创建失败" + err.Error())
	}
	return nil
}

func (u *UserService) Login(req request.LoginReq, refreshTime uint) (model.User, string, error) {
	user, err := u.Repo.FindUserByAccount(req.Account)
	if err != nil {
		return model.User{}, "", errors.New("获取用户信息失败")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return model.User{}, "", errors.New("密码错误")
	}
	refreshToken, err := u.SaveRefreshToken(user.ID, req.Device, refreshTime)
	if err != nil {
		return model.User{}, "", err
	}
	return user, refreshToken, nil
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
func (u *UserService) GetUserList(keyword string, department string, role string, page int, pageSize int) ([]model.UserInfo, int64, error) {
	if pageSize == 0 {
		pageSize = 20
	}
	if page == 0 {
		page = 1
	}
	userList, total, err := u.Repo.GetUserList(keyword, department, role, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	return userList, total, nil

}

func (u *UserService) UpdatePassword(id uint64, oldPassword string, newPassword1 string, newPassword2 string) error {
	if newPassword1 != newPassword2 {
		return errors.New("新密码两次不一致")
	}
	user, err := u.Repo.FindUserById(id)
	if err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword)); err != nil {
		return errors.New("旧密码错误")
	}
	password, err := bcrypt.GenerateFromPassword([]byte(newPassword1), bcrypt.DefaultCost)
	err = u.Repo.ResetPassword(id, string(password))
	if err != nil {
		return err
	}
	return nil
}
func (u *UserService) UpdateUserInfo(id uint64, username string) error {
	err := u.Repo.UpdateUsername(id, username)
	if err != nil {
		return err
	}
	return nil
}

func (u *UserService) SendVerificationCode(id uint64, types string) error {
	user, err := u.Repo.FindUserById(id)
	if err != nil {
		return err
	}
	if user.Email == "" {
		return errors.New("请先绑定邮箱")
	}
	if u.Repo.CheckCooldown(id, context.Background()) {
		return errors.New("间隔太短")
	}
	code := u.Email.NewVerificationCode(6)
	expire := u.Email.GetExpireTime()
	err = u.Repo.SaveVerificationCode(id, code, types, expire, context.Background())
	if err != nil {
		return err
	}
	err = u.Email.SendVerificationCode(user.Email, code)
	if err != nil {
		return err
	}
	err = u.Repo.SetCooldown(id, u.Email.Cooldown, context.Background())
	if err != nil {
		return err
	}
	return nil
}

func (u *UserService) ResetPassword(id uint64, code string, types string) error {
	verificationCode, err := u.Repo.GetVerificationCode(id, types, context.Background())
	if err != nil {
		return err
	}
	if verificationCode != code {
		return errors.New("验证码错误或者失效")
	}
	password, err := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	err = u.Repo.ResetPassword(id, string(password))
	if err != nil {
		return err
	}
	return nil
}
