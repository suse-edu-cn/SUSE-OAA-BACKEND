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

func (u *UserService) BatchUserInfo(req []request.BatchUserInfoReq, departmentID uint64, roleID uint64) ([]model.BatchUserInfo, error) {
	var update []model.UpdateUserItems
	var res []model.BatchUserInfo
	status := make(map[uint64]bool)
	level, err := u.RoleRepo.GetLevelByName("副会长")
	if err != nil {
		return res, err
	}
	errorItems := make(map[uint64]model.UpdateUserItems)
	errorMessage := make(map[uint64]string)
	departmentMap, err := u.DepartmentRepo.GetDepartmentMap()
	if err != nil {
		return res, errors.New("获取部门map失败")
	}
	roleMap, err := u.RoleRepo.GetRoleMap()
	if err != nil {
		return res, errors.New("获取职位map失败")
	}
	if roleMap[roleID] == nil {
		return res, errors.New("操作者没有权限")
	}
	var userIdList []uint64
	//区分出是否能改
	for i := 0; i < len(req); i++ {
		if status[req[i].UserID] {
			continue
		}
		status[req[i].UserID] = true
		if roleMap[req[i].RoleID] == nil || departmentMap[req[i].DepartmentID] == nil {
			userIdList = append(userIdList, req[i].UserID)
			tempItem := model.UpdateUserItems{}
			tempItem.UserID = req[i].UserID
			if departmentMap[req[i].DepartmentID] != nil {
				tempItem.DepartmentID = &req[i].DepartmentID
			} else {
				errorMessage[req[i].UserID] = "部门不存在"
			}
			if roleMap[req[i].RoleID] != nil {
				tempItem.RoleID = &req[i].RoleID
			} else {
				if errorMessage[req[i].UserID] == "" {
					errorMessage[req[i].UserID] = "职位不存在"
				} else {
					errorMessage[req[i].UserID] = "部门，职位都不存在"
				}

			}
			errorItems[req[i].UserID] = tempItem
		} else if (roleMap[roleID].Level >= level || //副会长以及以上
			(roleMap[roleID].Level > roleMap[req[i].RoleID].Level && //职位比要设置的职位大
				departmentMap[departmentID] != nil &&
				departmentID == req[i].DepartmentID)) && //同一部门
			(level > roleMap[req[i].RoleID].Level) { //只能改副会长以下职位
			update = append(update, model.UpdateUserItems{
				UserID:       req[i].UserID,
				DepartmentID: &req[i].DepartmentID,
				RoleID:       &req[i].RoleID,
			})
		} else {
			userIdList = append(userIdList, req[i].UserID)
			errorItems[req[i].UserID] = model.UpdateUserItems{
				UserID:       req[i].UserID,
				DepartmentID: &req[i].DepartmentID,
				RoleID:       &req[i].RoleID,
			}
			if level <= roleMap[req[i].RoleID].Level {
				errorMessage[req[i].UserID] = "不能修改成该职位"
			} else {
				errorMessage[req[i].UserID] = "修改人权限不够"
			}
		}
	}
	//将修改存入数据库
	err = u.Repo.BatchUserInfoList(context.Background(), update)
	if err != nil {
		return res, errors.New("存入修改失败")
	}
	//拼接味修改的数据
	usersInfo, err := u.Repo.GetUsersInfo(userIdList)
	if err != nil {
		return res, err
	}
	for _, id := range userIdList {
		temp := model.BatchUserInfo{
			StudentID:    usersInfo[id].StudentID,
			Name:         usersInfo[id].Name,
			Department:   usersInfo[id].Department,
			Role:         usersInfo[id].Role,
			Username:     usersInfo[id].Username,
			ErrorMessage: errorMessage[id],
		}
		if errorItems[id].RoleID != nil {
			temp.ToRole = roleMap[*errorItems[id].RoleID].Name
		}
		if errorItems[id].DepartmentID != nil {
			temp.ToDepartment = departmentMap[*errorItems[id].DepartmentID].Name
		}
		res = append(res, temp)
	}
	return res, nil
}
