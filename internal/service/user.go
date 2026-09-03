package service

import (
	"context"
	"errors"
	"log"
	"strings"
	"suseoaa/internal/storage"
	"suseoaa/pkg/utils"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"suseoaa/internal/model"
	"suseoaa/internal/repository"
	"suseoaa/internal/request"
)

type UserService struct {
	Repo           repository.UserRepository
	RoleRepo       repository.RoleRepository
	DepartmentRepo repository.DepartmentRepository
	Email          EmailService
	File           FileService
}

func NewUserService(
	repo repository.UserRepository,
	roleRepo repository.RoleRepository,
	departmentRepo repository.DepartmentRepository,
	email EmailService,
	file FileService) UserService {
	return UserService{
		Repo:           repo,
		RoleRepo:       roleRepo,
		DepartmentRepo: departmentRepo,
		Email:          email,
		File:           file,
	}
}

func (u *UserService) Register(req request.RegisterReq) error {
	req.StudentID = strings.TrimSpace(req.StudentID)
	req.Username = strings.TrimSpace(req.Username)
	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(req.Email)

	if req.StudentID == "" || req.Username == "" || req.Name == "" || req.Email == "" || req.Password == "" {
		return errors.New("注册信息不能为空")
	}

	// 传入 req.Username 校验
	if err := u.Repo.CheckExist(req.StudentID, req.Email, req.Username); err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("密码加密失败")
	}

	role, err := u.RoleRepo.FindByName("会员")
	if err != nil {
		return errors.New("获取默认角色失败: " + err.Error())
	}
	department, err := u.DepartmentRepo.GetDepartmentByName("开放原子开源协会")
	if err != nil {
		return errors.New("获取默认部门失败" + err.Error())
	}
	user := model.User{
		StudentID:  req.StudentID,
		Username:   req.Username,
		Name:       req.Name,
		Email:      req.Email,
		Password:   string(hash),
		Role:       role,
		Department: department,
	}

	if err := u.Repo.CreateUser(user); err != nil {
		return errors.New("创建用户失败: " + err.Error())
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

func (u *UserService) GetUserInfo(ctx context.Context, id uint64) (model.UserInfo, error) {
	info, err := u.Repo.GetUserInfoById(id)
	if err != nil {
		return model.UserInfo{}, errors.New("查询失败" + err.Error())
	}

	avatar, err := u.getAvatarURL(ctx, info.Avatar)
	if err != nil {
		return model.UserInfo{}, errors.New("生成头像链接失败" + err.Error())
	}
	info.Avatar = avatar
	return info, nil
}
func (u *UserService) GetDepartmentIDAndRoleIDByID(id uint64) (uint64, uint64, error) {
	return u.Repo.GetDepartmentIDAndRoleIDByID(id)
}

func (u *UserService) getAvatarURL(ctx context.Context, avatar string) (string, error) {
	const defaultAvatar = "avatar/default.png"

	if avatar != "" {
		if _, err := u.File.Storage.GetFileInfo(ctx, avatar); err == nil {
			if url, err := u.File.Storage.GeneratePresignedURL(ctx, avatar); err == nil {
				return url, nil
			}
		}
	}

	return u.File.Storage.GeneratePresignedURL(ctx, defaultAvatar)
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
	const (
		defaultPageSize = 20
		maxPageSize     = 100
	)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	userList, total, err := u.Repo.GetUserList(keyword, department, role, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	return userList, total, nil

}
func (u *UserService) FindUserByID(id uint64) (model.User, error) {
	return u.Repo.FindUserById(id)
}

func (u *UserService) UpdatePassword(id uint64, oldPassword string, newPassword string) error {
	user, err := u.Repo.FindUserById(id)
	if err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword)); err != nil {
		return errors.New("旧密码错误")
	}
	password, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("密码加密失败")
	}
	err = u.Repo.ResetPassword(id, string(password))
	if err != nil {
		return err
	}
	return nil
}
func (u *UserService) UpdateUserInfo(ctx context.Context, id uint64, username string, email string, avatar string) error {
	size, err := u.File.Storage.GetFileInfo(ctx, avatar)
	if err != nil {
		return errors.New("头像不存在" + err.Error())
	}
	if size > storage.MaxImageSize {
		return errors.New("头像体积过大")
	}
	user, err := u.Repo.FindUserById(id)
	if err != nil {
		return err
	}

	const defaultAvatar = "avatar/default.png"
	oldAvatar := user.Avatar

	err = u.Repo.UpdateUser(id, username, email, avatar)
	if err != nil {
		return err
	}

	if oldAvatar == "" || oldAvatar == avatar || oldAvatar == defaultAvatar {
		return nil
	}

	if err = u.File.Storage.DeleteFile(ctx, oldAvatar); err != nil {
		// 头像更新已经成功，旧头像清理失败不应影响主流程。
		// 这里保留日志级别的错误信息，方便后续排查 MinIO/对象路径问题。
		log.Printf("删除旧头像失败, user_id=%d, avatar=%s, err=%v", id, oldAvatar, err)
	}
	return nil
}

func (u *UserService) SendVerificationCode(account string, types string) error {
	user, err := u.Repo.FindUserByAccount(account)
	if err != nil {
		return err
	}
	if user.Email == "" {
		return errors.New("请先绑定邮箱")
	}
	if cooldown, err := u.Repo.CheckCooldown(user.ID, context.Background()); err != nil {
		return err
	} else if cooldown {
		return errors.New("间隔太短")
	}
	code := u.Email.NewVerificationCode(6)
	expire := u.Email.GetExpireTime()
	if err := u.Repo.SaveVerificationCode(user.ID, code, types, expire, context.Background()); err != nil {
		return err
	}
	if err := u.Email.SendVerificationCode(user.Email, code); err != nil {
		return err
	}
	if err := u.Repo.SetCooldown(user.ID, u.Email.Cooldown, context.Background()); err != nil {
		return err
	}
	return nil
}

func (u *UserService) ResetPassword(account string, code string, types string) error {
	user, err := u.Repo.FindUserByAccount(account)
	if err != nil {
		return err
	}

	verificationCode, err := u.Repo.GetVerificationCode(user.ID, types, context.Background())
	if err != nil {
		return err
	}
	if verificationCode != code {
		return errors.New("验证码错误或者失效")
	}
	if err := u.Repo.DeleteVerificationCode(user.ID, types, context.Background()); err != nil {
		return err
	}
	password, err := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := u.Repo.ResetPassword(user.ID, string(password)); err != nil {
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
	errorMessage := make(map[uint64]string)
	departmentMap, err := u.DepartmentRepo.GetDepartmentMap()
	if err != nil {
		return res, errors.New("获取部门map失败")
	}
	roleMap, err := u.RoleRepo.GetRoleMap()
	if err != nil {
		return res, errors.New("获取职位map失败")
	}
	operatorRole := roleMap[roleID]
	operatorDepartment := departmentMap[departmentID]
	if operatorRole == nil || !operatorRole.IsActive ||
		operatorDepartment == nil || !operatorDepartment.IsActive {
		return res, errors.New("操作者的部门或职位已停用")
	}
	var userIdList []uint64
	//区分出是否能改
	for i := range req {
		if status[req[i].UserID] {
			continue
		}
		status[req[i].UserID] = true
		if _, err := u.Repo.FindUserById(req[i].UserID); err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return res, errors.New("查询目标用户失败")
			}
			userIdList = append(userIdList, req[i].UserID)
			errorMessage[req[i].UserID] = "用户不存在"
			continue
		}
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
		} else if err := u.VerifyDepartmentPosition(req[i].DepartmentID, req[i].RoleID); err != nil { //职位和部门不合理
			userIdList = append(userIdList, req[i].UserID)
			errorMessage[req[i].UserID] = err.Error()

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
	//拼接未修改的数据
	usersInfo, err := u.Repo.GetUsersInfo(userIdList)
	if err != nil {
		return res, err
	}
	for _, id := range userIdList {
		temp := model.BatchUserInfo{
			ID:           id,
			StudentID:    usersInfo[id].StudentID,
			Name:         usersInfo[id].Name,
			Username:     usersInfo[id].Username,
			ErrorMessage: errorMessage[id],
		}
		res = append(res, temp)
	}
	return res, nil
}
func (u *UserService) VerifyDepartmentPosition(departmentID uint64, roleID uint64) error {
	department, err := u.DepartmentRepo.GetDepartmentByID(departmentID)
	if err != nil {
		return err
	}
	role, err := u.RoleRepo.GetRoleByID(roleID)
	if err != nil {
		return err
	}
	if !department.IsActive {
		return errors.New("部门已停用")
	}
	if !role.IsActive {
		return errors.New("职位已停用")
	}
	if department.Type != role.Type {
		return errors.New("部门和职位不匹配")
	}
	return nil
}
func (u *UserService) DeleteUser(id uint64, userID uint64) error {
	level, _, err := u.Repo.GetActiveRoleLevelAndDepartment(id)
	if err != nil {
		return err
	}
	userLevel, _, err := u.Repo.GetRoleLevelAndDepartment(userID)
	if err != nil {
		return err
	}
	if level < 80 || userLevel >= level {
		return errors.New("权限不够")
	}
	err = u.Repo.DeleteUserByID(userID)
	if err != nil {
		return err
	}
	return nil
}
