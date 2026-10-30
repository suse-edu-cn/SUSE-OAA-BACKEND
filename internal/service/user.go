package service

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"suseoaa/pkg/logger"
	"suseoaa/pkg/utils"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"suseoaa/internal/model"
	"suseoaa/internal/repository"
	"suseoaa/internal/request"
)

type UserService struct {
	Repo           *repository.UserRepository
	RoleRepo       *repository.RoleRepository
	DepartmentRepo *repository.DepartmentRepository
	Email          *EmailService
	File           *FileService
}

func NewUserService(
	repo *repository.UserRepository,
	roleRepo *repository.RoleRepository,
	departmentRepo *repository.DepartmentRepository,
	email *EmailService,
	file *FileService) *UserService {
	return &UserService{
		Repo:           repo,
		RoleRepo:       roleRepo,
		DepartmentRepo: departmentRepo,
		Email:          email,
		File:           file,
	}
}

func (u *UserService) Register(ctx context.Context, req request.RegisterReq) error {
	req.StudentID = strings.TrimSpace(req.StudentID)
	req.Username = strings.TrimSpace(req.Username)
	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(req.Email)

	if req.StudentID == "" || req.Username == "" || req.Name == "" || req.Email == "" || req.Password == "" {
		return errors.New("注册信息不能为空")
	}
	if !(len(req.StudentID) == 11 || (len(req.StudentID) == 12 && req.StudentID[0] == '3')) {
		return errors.New("学号格式错误")
	}
	if req.Grade <= 2000 || req.Grade > 9999 {
		return errors.New("入学年份不合法")
	}

	if err := u.Repo.CheckExist(ctx, req.StudentID, req.Email, req.Username); err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("密码加密失败")
	}

	role, err := u.RoleRepo.FindByName(ctx, "会员")
	if err != nil {
		return errors.New("获取默认角色失败: " + err.Error())
	}
	department, err := u.DepartmentRepo.GetDepartmentByName(ctx, "开放原子开源协会")
	if err != nil {
		return errors.New("获取默认部门失败" + err.Error())
	}
	user := model.User{
		StudentID:   req.StudentID,
		Username:    req.Username,
		Name:        req.Name,
		Email:       req.Email,
		Password:    string(hash),
		Role:        role,
		Department:  department,
		Grade:       req.Grade,
		IsGraduated: false,
	}

	if err := u.Repo.CreateUser(ctx, user); err != nil {
		return errors.New("创建用户失败: " + err.Error())
	}
	return nil
}

func (u *UserService) Login(ctx context.Context, req request.LoginReq, refreshTime uint) (model.User, string, error) {
	user, err := u.Repo.FindUserByAccount(ctx, req.Account)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.User{}, "", errors.New("账号不存在")
		}
		return model.User{}, "", err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return model.User{}, "", errors.New("密码错误")
	}
	refreshToken, err := u.SaveRefreshToken(ctx, user.ID, req.Device, refreshTime)
	if err != nil {
		return model.User{}, "", err
	}
	return user, refreshToken, nil
}

func (u *UserService) GetUserInfo(ctx context.Context, id uint64) (model.UserInfo, error) {
	info, err := u.Repo.GetUserInfoById(ctx, id)
	if err != nil {
		return model.UserInfo{}, errors.New("查询失败" + err.Error())
	}

	avatar, err := u.getAvatarURL(ctx, info.Avatar.URI)
	if err != nil {
		return model.UserInfo{}, errors.New("生成头像链接失败" + err.Error())
	}
	info.Avatar.URL = avatar
	return info, nil
}

func (u *UserService) GetDepartmentIDAndRoleIDByID(ctx context.Context, id uint64) (uint64, uint64, error) {
	return u.Repo.GetDepartmentIDAndRoleIDByID(ctx, id)
}

func (u *UserService) getAvatarURL(ctx context.Context, avatar string) (string, error) {
	const defaultAvatar = "avatar/default.png"

	targetAvatar := strings.TrimSpace(avatar)
	if targetAvatar == "" {
		targetAvatar = defaultAvatar
	}

	if u.File == nil || u.File.ImgStorage == nil {
		return targetAvatar, nil
	}

	url, err := u.File.ImgStorage.GeneratePresignedURL(ctx, targetAvatar)
	if err != nil {
		if targetAvatar != defaultAvatar {
			return u.File.ImgStorage.GeneratePresignedURL(ctx, defaultAvatar)
		}
		return "", err
	}

	return url, nil
}

func (u *UserService) SaveRefreshToken(ctx context.Context, id uint64, device string, time uint) (string, error) {
	token, err := utils.GetUUID()
	if err != nil {
		return "", errors.New("生成refresh_token失败")
	}
	err = u.Repo.SaveRefreshToken(ctx, id, device, token, time)
	if err != nil {
		return "", err
	}
	return token, nil
}

func (u *UserService) DeleteRefreshToken(ctx context.Context, id uint64, device string) error {
	err := u.Repo.DeleteRefreshToken(ctx, id, device)
	if err != nil {
		return err
	}
	return nil
}

func (u *UserService) GetRefreshToken(ctx context.Context, id uint64, device string) (string, error) {
	token, err := u.Repo.GetRefreshToken(ctx, id, device)
	if err != nil {
		return "", err
	}
	return token, nil
}

func (u *UserService) GetRefreshTokenByToken(ctx context.Context, token string) (*model.RefreshToken, error) {
	return u.Repo.GetRefreshTokenByToken(ctx, token)
}

func (u *UserService) GetUserList(ctx context.Context, keyword string, department string, role string, departmentID uint64, roleID uint64, grade uint, isGraduated *bool, page int, pageSize int, isAll *bool, userID uint64) ([]model.UserInfo, int64, error) {
	user, err := u.Repo.FindUserById(ctx, userID)
	if err != nil {
		return nil, 0, errors.New("获取用户职位失败")
	}
	if user.IsGraduated {
		return nil, 0, errors.New("已毕业成员无操作权限")
	}
	level, _, err := u.Repo.GetActiveRoleLevelAndDepartment(ctx, userID)
	if err != nil {
		return nil, 0, errors.New("权限不够")
	}
	if level < 50 {
		return nil, 0, errors.New("权限不够")
	}
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
	userList, total, err := u.Repo.GetUserList(ctx, keyword, department, role, departmentID, roleID, grade, isGraduated, page, pageSize, isAll)
	if err != nil {
		return nil, 0, err
	}
	for index := range userList {
		url, err := u.getAvatarURL(ctx, userList[index].Avatar.URI)
		if err != nil {
			continue
		}
		userList[index].Avatar.URL = url
	}
	return userList, total, nil
}

func (u *UserService) FindUserByID(ctx context.Context, id uint64) (model.User, error) {
	return u.Repo.FindUserById(ctx, id)
}

func (u *UserService) UpdatePassword(ctx context.Context, id uint64, oldPassword string, newPassword string) error {
	user, err := u.Repo.FindUserById(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("用户不存在")
		}
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword)); err != nil {
		return errors.New("旧密码错误")
	}
	password, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("密码加密失败")
	}
	err = u.Repo.ResetPassword(ctx, id, string(password))
	if err != nil {
		return err
	}
	return nil
}

func (u *UserService) UpdateUserInfo(ctx context.Context, id uint64, username string, email string, avatar string, grade uint) error {
	if grade != 0 && (grade <= 2000 || grade > 9999) {
		return errors.New("入学年份不合法")
	}
	avatar = strings.TrimSpace(avatar)
	const defaultAvatar = "avatar/default.png"
	if avatar != "" && avatar != defaultAvatar {
		if !strings.HasPrefix(avatar, "avatar/") {
			return errors.New("头像路径错误")
		}
		if strings.Contains(avatar, "..") {
			return errors.New("头像路径不合法")
		}
		ext := strings.ToLower(filepath.Ext(avatar))
		if !allowedExt[ext] {
			return errors.New("头像类型错误")
		}
	}
	user, err := u.Repo.FindUserById(ctx, id)
	if err != nil {
		return err
	}
	oldAvatar := user.Avatar

	err = u.Repo.UpdateUser(ctx, id, username, email, avatar, grade)
	if err != nil {
		return err
	}

	if oldAvatar == "" || oldAvatar == avatar || oldAvatar == defaultAvatar {
		return nil
	}

	if err = u.File.ImgStorage.DeleteFile(ctx, oldAvatar); err != nil {
		logger.WarnContext(ctx, "删除旧头像失败", "user_id", id, "avatar", oldAvatar, "err", err)
	}
	return nil
}

func (u *UserService) SendVerificationCode(ctx context.Context, account string, types string) error {
	user, err := u.Repo.FindUserByAccount(ctx, account)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("账号不存在")
		}
		return err
	}
	if user.Email == "" {
		return errors.New("请先绑定邮箱")
	}
	if types == "cancel_delete" {
		if user.ScheduledDeleteAt == nil {
			return errors.New("当前账号未处于注销冷静期")
		}
		if time.Now().After(*user.ScheduledDeleteAt) {
			return errors.New("注销冷静期已结束")
		}
	}
	if types == "delete_user" {
		if user.ScheduledDeleteAt != nil && time.Now().Before(*user.ScheduledDeleteAt) {
			return errors.New("账号已处于注销冷静期")
		}
	}
	token, err := utils.GetUUID()
	if err != nil {
		token = fmt.Sprintf("%d-%d", user.ID, time.Now().UnixNano())
	}
	acquired, err := u.Repo.SetCooldownNX(ctx, user.ID, token, u.Email.Cooldown)
	if err != nil {
		return err
	}
	if !acquired {
		return errors.New("间隔太短")
	}

	code := u.Email.NewVerificationCode(6)
	expire := u.Email.GetExpireTime()
	if err := u.Repo.SaveVerificationCode(ctx, user.ID, code, types, expire); err != nil {
		_ = u.Repo.DeleteCooldown(ctx, token, user.ID)
		return err
	}
	if err := u.Email.SendVerificationCode(user.Email, code); err != nil {
		_ = u.Repo.DeleteCooldown(ctx, token, user.ID)
		_ = u.Repo.DeleteVerificationCode(ctx, user.ID, types)
		return err
	}
	return nil
}

func (u *UserService) ResetPassword(ctx context.Context, account string, code string, resetPassword string) error {
	user, err := u.Repo.FindUserByAccount(ctx, account)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("账号不存在")
		}
		return err
	}
	if err := u.Repo.VerifyAndConsumeVerificationCode(ctx, user.ID, "reset_password", code, 5); err != nil {
		return err
	}

	password, err := bcrypt.GenerateFromPassword([]byte(resetPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := u.Repo.ResetPassword(ctx, user.ID, string(password)); err != nil {
		return err
	}
	return nil
}

func (u *UserService) BatchUserInfo(ctx context.Context, req []request.BatchUserInfoReq, departmentID uint64, roleID uint64) ([]model.BatchUserInfo, error) {
	var update []model.UpdateUserItems
	var res []model.BatchUserInfo
	status := make(map[uint64]bool)
	level, err := u.RoleRepo.GetLevelByName(ctx, "副会长")
	if err != nil {
		return res, err
	}
	errorMessage := make(map[uint64]string)
	departmentMap, err := u.DepartmentRepo.GetDepartmentMap(ctx)
	if err != nil {
		return res, errors.New("获取部门map失败")
	}
	roleMap, err := u.RoleRepo.GetRoleMap(ctx)
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
		targetUser, err := u.Repo.FindUserById(ctx, req[i].UserID)
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return res, errors.New("查询目标用户失败")
			}
			userIdList = append(userIdList, req[i].UserID)
			errorMessage[req[i].UserID] = "用户不存在"
			continue
		}

		targetCurrentRole := roleMap[targetUser.RoleID]
		targetCurrentLevel := uint64(0)
		if targetCurrentRole != nil {
			targetCurrentLevel = targetCurrentRole.Level
		}

		// 1. 不能修改同级或更高职位的用户
		if targetCurrentLevel >= operatorRole.Level {
			userIdList = append(userIdList, req[i].UserID)
			errorMessage[req[i].UserID] = "不能修改同级或更高职位的用户"
			continue
		}

		// 2. 低于副会长的管理者（如部长），不能修改原本不属于本部门的用户
		if operatorRole.Level < level && targetUser.DepartmentID != departmentID {
			userIdList = append(userIdList, req[i].UserID)
			errorMessage[req[i].UserID] = "不能跨部门修改其他部门的用户"
			continue
		}

		// 3. 年级修改权限锁：只有副会长及以上（Level >= 80）可操作
		if req[i].Grade != nil {
			if operatorRole.Level < level {
				userIdList = append(userIdList, req[i].UserID)
				errorMessage[req[i].UserID] = "权限不足：只有副会长及以上管理才可修改年级"
				continue
			}
		}

		// 4. 入学年份合法性校验（大于 2000 且为四位数）
		if req[i].Grade != nil && (*req[i].Grade <= 2000 || *req[i].Grade > 9999) {
			userIdList = append(userIdList, req[i].UserID)
			errorMessage[req[i].UserID] = "入学年份不合法"
			continue
		}

		// 5. 已毕业成员不可任命在任职务
		if targetUser.IsGraduated {
			if targetCurrentRole == nil || targetCurrentRole.Name != "开发者" {
				if roleMap[req[i].RoleID] != nil && roleMap[req[i].RoleID].Name != "会员" {
					userIdList = append(userIdList, req[i].UserID)
					errorMessage[req[i].UserID] = "已毕业成员不可任命在任职务"
					continue
				}
			}
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
		} else if err := u.VerifyDepartmentPosition(ctx, req[i].DepartmentID, req[i].RoleID); err != nil { //职位和部门不合理
			userIdList = append(userIdList, req[i].UserID)
			errorMessage[req[i].UserID] = err.Error()

		} else if (operatorRole.Level >= level || //副会长以及以上
			(operatorRole.Level > roleMap[req[i].RoleID].Level && //职位比要设置的职位大
				departmentMap[departmentID] != nil &&
				departmentID == req[i].DepartmentID)) && //同一部门
			(level > roleMap[req[i].RoleID].Level) { //只能改副会长以下职位
			update = append(update, model.UpdateUserItems{
				UserID:       req[i].UserID,
				DepartmentID: &req[i].DepartmentID,
				RoleID:       &req[i].RoleID,
				Grade:        req[i].Grade,
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
	err = u.Repo.BatchUserInfoList(ctx, update)
	if err != nil {
		return res, errors.New("存入修改失败")
	}
	//拼接未修改的数据
	usersInfo, err := u.Repo.GetUsersInfo(ctx, userIdList)
	if err != nil {
		return res, err
	}
	for _, id := range userIdList {
		temp := model.BatchUserInfo{
			UserID:       id,
			StudentID:    usersInfo[id].StudentID,
			Name:         usersInfo[id].Name,
			Username:     usersInfo[id].Username,
			ErrorMessage: errorMessage[id],
		}
		res = append(res, temp)
	}
	if len(res) == 0 {
		return res, nil
	}
	return res, errors.New("存在部分错误")
}

func (u *UserService) VerifyDepartmentPosition(ctx context.Context, departmentID uint64, roleID uint64) error {
	department, err := u.DepartmentRepo.GetDepartmentByID(ctx, departmentID)
	if err != nil {
		return err
	}
	role, err := u.RoleRepo.GetRoleByID(ctx, roleID)
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

// UserDeletionGracePeriod 账户注销冷静期常量（目前为 1 天）
const UserDeletionGracePeriod = 24 * time.Hour

func (u *UserService) DeleteUser(ctx context.Context, id uint64, userID uint64, code string) (*time.Time, error) {
	// 互斥校验：两个字段不能同时有参数
	if userID != 0 && code != "" {
		return nil, errors.New("参数错误：user_id 与 code 不能同时传递")
	}

	// 场景 1：仅传递 user_id（此时 code 为空） -> 管理员删除其他用户
	if userID != 0 {
		if userID == id {
			return nil, errors.New("注销本人账号请提供验证码，或留空查询冷静期")
		}

		level, _, err := u.Repo.GetActiveRoleLevelAndDepartment(ctx, id)
		if err != nil {
			return nil, err
		}
		userLevel, _, err := u.Repo.GetRoleLevelAndDepartment(ctx, userID)
		if err != nil {
			return nil, err
		}
		if level < 80 || userLevel >= level {
			return nil, errors.New("权限不够")
		}

		// 管理员直接删除其他用户（直接执行物理软删除与Token级联清理，不设冷静期）
		if err := u.Repo.DeleteUserByID(ctx, userID); err != nil {
			return nil, err
		}
		return nil, nil
	}

	// 场景 2 & 3：本人操作（userID 为 0）
	user, err := u.Repo.FindUserById(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("用户不存在")
		}
		return nil, err
	}

	now := time.Now()
	// 已经在冷静期内：检查倒计时（无论传了 code 还是两个都空，都返回冷静期倒计时）
	if user.ScheduledDeleteAt != nil {
		remaining := user.ScheduledDeleteAt.Sub(now)
		if remaining > 0 {
			// 返回冷静期截止时间，供 Handler 返回 400 及注销时间
			return user.ScheduledDeleteAt, nil
		}

		// 冷静期倒计时已结束：立即彻底注销删除
		if err := u.Repo.DeleteUserByID(ctx, id); err != nil {
			return nil, err
		}
		return nil, nil
	}

	// 未在冷静期内：
	// 场景 3：两个都为空（userID == 0 且 code == ""） -> 查询冷静期，但账号并未处于冷静期
	if code == "" {
		return nil, errors.New("当前账号未处于注销冷静期")
	}

	// 场景 2：仅传递 code（首次申请注销） -> 校验验证码并进入冷静期
	const scene = "delete_user"
	if err := u.Repo.VerifyAndConsumeVerificationCode(ctx, id, scene, code, 5); err != nil {
		return nil, err
	}
	scheduledAt := now.Add(UserDeletionGracePeriod)
	if err := u.Repo.SetScheduledDeleteAt(ctx, id, &scheduledAt); err != nil {
		return nil, errors.New("设置注销冷静期失败: " + err.Error())
	}
	return nil, nil
}

func (u *UserService) CancelDeleteUser(ctx context.Context, userID uint64, code string) error {
	user, err := u.Repo.FindUserById(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("用户不存在")
		}
		return errors.New("获取用户信息失败: " + err.Error())
	}
	if user.ScheduledDeleteAt == nil {
		return errors.New("当前账号未处于注销冷静期")
	}
	if time.Now().After(*user.ScheduledDeleteAt) {
		return errors.New("注销冷静期已结束，无法取消")
	}

	const scene = "cancel_delete"
	if err := u.Repo.VerifyAndConsumeVerificationCode(ctx, userID, scene, code, 5); err != nil {
		return err
	}

	if err := u.Repo.SetScheduledDeleteAt(ctx, userID, nil); err != nil {
		return errors.New("取消注销冷静期失败: " + err.Error())
	}
	return nil
}

func (u *UserService) StartUserDeletionExecutor(ctx context.Context) {
	u.executeDueUserDeletionsAndLog(ctx)

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			u.executeDueUserDeletionsAndLog(ctx)
		}
	}
}

func (u *UserService) ExecuteDueUserDeletions(ctx context.Context) (int, error) {
	now := time.Now()
	users, err := u.Repo.GetDueScheduledDeleteUsers(ctx, now)
	if err != nil {
		return 0, err
	}

	deleted := 0
	var errs []error
	for _, user := range users {
		if err := u.Repo.DeleteUserByID(ctx, user.ID); err != nil {
			errs = append(errs, fmt.Errorf("user_id=%d 注销失败: %w", user.ID, err))
			continue
		}
		deleted++
	}

	return deleted, errors.Join(errs...)
}

func (u *UserService) executeDueUserDeletionsAndLog(ctx context.Context) {
	deleted, err := u.ExecuteDueUserDeletions(ctx)
	if deleted > 0 {
		logger.InfoContext(ctx, "已自动执行注销到期用户数量", "deleted", deleted)
	}
	if err != nil {
		logger.ErrorContext(ctx, "执行到期用户自动注销失败", "err", err)
	}
}

// CalculateGraduationMaxGrade 计算给定时间对应的已毕业最大入学年份
// 规则：每年 6 月份及之后，入学满 4 年（即 grade <= now.Year() - 4）的用户毕业；
// 6 月份之前，入学满 4 年的为 grade <= now.Year() - 5。
func (u *UserService) CalculateGraduationMaxGrade(now time.Time) int {
	loc := time.FixedZone("Asia/Shanghai", 8*60*60)
	current := now.In(loc)
	if current.Month() >= time.June {
		return current.Year() - 4
	}
	return current.Year() - 5
}

func (u *UserService) StartUserGraduationExecutor(ctx context.Context) {
	u.executeDueUserGraduationsAndLog(ctx)

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			u.executeDueUserGraduationsAndLog(ctx)
		}
	}
}

func (u *UserService) ExecuteDueUserGraduations(ctx context.Context, now time.Time) (int, error) {
	maxGrade := u.CalculateGraduationMaxGrade(now)
	if maxGrade <= 0 {
		return 0, nil
	}

	users, err := u.Repo.GetDueGraduationUsers(ctx, uint(maxGrade))
	if err != nil {
		return 0, err
	}
	if len(users) == 0 {
		return 0, nil
	}

	roleMap, err := u.RoleRepo.GetRoleMap(ctx)
	if err != nil {
		return 0, errors.New("获取职位列表失败: " + err.Error())
	}
	departmentMap, err := u.DepartmentRepo.GetDepartmentMap(ctx)
	if err != nil {
		return 0, errors.New("获取部门列表失败: " + err.Error())
	}

	var defaultMemberRoleID uint64
	for id, r := range roleMap {
		if r.Name == "会员" {
			defaultMemberRoleID = id
			break
		}
	}
	var defaultDeptID uint64
	for id, d := range departmentMap {
		if d.Name == "开放原子开源协会" {
			defaultDeptID = id
			break
		}
	}
	if defaultMemberRoleID == 0 || defaultDeptID == 0 {
		return 0, errors.New("系统默认部门或默认角色未初始化")
	}

	var devIDs []uint64
	var nonDevIDs []uint64
	for _, user := range users {
		r := roleMap[user.RoleID]
		if r != nil && r.Name == "开发者" {
			devIDs = append(devIDs, user.ID)
		} else {
			nonDevIDs = append(nonDevIDs, user.ID)
		}
	}

	if err := u.Repo.GraduateUsers(ctx, nonDevIDs, devIDs, defaultMemberRoleID, defaultDeptID); err != nil {
		return 0, err
	}

	return len(users), nil
}

func (u *UserService) executeDueUserGraduationsAndLog(ctx context.Context) {
	graduated, err := u.ExecuteDueUserGraduations(ctx, time.Now())
	if graduated > 0 {
		logger.InfoContext(ctx, "已自动执行到期毕业用户数量", "graduated", graduated)
	}
	if err != nil {
		logger.ErrorContext(ctx, "执行到期毕业用户失败", "err", err)
	}
}
