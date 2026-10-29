package repository

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"suseoaa/internal/model"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// 仅当键的值与传入的 token 一致时才删除，防止超时误删他人后置标记
var releaseCooldownScript = redis.NewScript(`
if redis.call("get", KEYS[1]) == ARGV[1] then
    return redis.call("del", KEYS[1])
else
    return 0
end
`)

type UserRepository struct {
	DB  *gorm.DB
	Rdb *redis.Client
}

func NewUserRepository(db *gorm.DB, rdb *redis.Client) *UserRepository {
	return &UserRepository{
		DB:  db,
		Rdb: rdb,
	}
}

func (u *UserRepository) CreateUser(ctx context.Context, user model.User) error {
	if err := u.DB.WithContext(ctx).Create(&user).Error; err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return errors.New("学号、邮箱或用户名已被占用")
		}
		return err
	}
	return nil
}

func (u *UserRepository) FindUserByAccount(ctx context.Context, message string) (model.User, error) {
	var user model.User
	err := u.DB.WithContext(ctx).Model(&model.User{}).Where("username = ? or email = ? or student_id = ? ", message, message, message).First(&user).Error
	return user, err
}

func (u *UserRepository) CheckExist(ctx context.Context, studentId string, email string, username string) error {
	var num int64
	err := u.DB.WithContext(ctx).Model(&model.User{}).
		Where("email = ? OR student_id = ? OR username = ?", email, studentId, username).
		Count(&num).Error
	if err != nil {
		return errors.New("检查用户唯一性失败: " + err.Error())
	}
	if num > 0 {
		return errors.New("学号、邮箱或用户名已被占用")
	}
	return nil
}

func (u *UserRepository) FindUserById(ctx context.Context, id uint64) (model.User, error) {
	var user model.User
	err := u.DB.WithContext(ctx).Where("id = ?", id).First(&user).Error
	return user, err
}

func (u *UserRepository) ResetPassword(ctx context.Context, id uint64, password string) error {
	err := u.DB.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Update("password", password).Error
	if err != nil {
		return errors.New("更新失败")
	}
	return nil
}

func (u *UserRepository) UpdateUser(ctx context.Context, id uint64, username string, email string, avatar string, grade uint) error {
	updateData := map[string]any{
		"username": username,
		"email":    email,
		"avatar":   avatar,
	}
	if grade != 0 {
		updateData["grade"] = grade
	}
	err := u.DB.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Updates(updateData).Error
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return errors.New("邮箱或用户名已被占用")
		}
		return errors.New("更新失败")
	}
	return nil
}

func (u *UserRepository) GetUserInfoById(ctx context.Context, id uint64) (model.UserInfo, error) {
	var user model.User
	var info model.UserInfo
	err := u.DB.WithContext(ctx).Preload("Department").Preload("Role").First(&user, id).Error
	if err != nil {
		return info, err
	}
	if user.Role != nil {
		info.Role = user.Role.Name
		info.RoleLevel = user.Role.Level
	}
	if user.Department != nil {
		info.Department = user.Department.Name
	}
	info.UserID = id
	info.Email = user.Email
	info.Username = user.Username
	info.Name = user.Name
	info.StudentID = user.StudentID
	info.Avatar.URI = user.Avatar
	info.Grade = user.Grade
	info.IsGraduated = user.IsGraduated
	now := time.Now()
	if user.ScheduledDeleteAt != nil && now.Before(*user.ScheduledDeleteAt) {
		info.ScheduledDeleteAt = user.ScheduledDeleteAt.In(time.FixedZone("Asia/Shanghai", 8*60*60)).Format("2006-01-02 15:04:05")
		info.IsDeleting = true
	}
	return info, nil
}

func (u *UserRepository) GetRoleLevelAndDepartment(ctx context.Context, id uint64) (uint64, string, error) {
	var user model.User
	err := u.DB.WithContext(ctx).Preload("Department").Preload("Role").Where("id = ?", id).First(&user).Error
	if err != nil {
		return 0, "", err
	}
	if user.Role == nil {
		return 0, "", errors.New("角色不存在")
	}
	deptName := ""
	if user.Department != nil {
		deptName = user.Department.Name
	}
	return user.Role.Level, deptName, nil
}

func (u *UserRepository) GetActiveRoleLevelAndDepartment(ctx context.Context, id uint64) (uint64, string, error) {
	var user model.User
	err := u.DB.WithContext(ctx).Preload("Department").Preload("Role").Where("id = ?", id).First(&user).Error
	if err != nil {
		return 0, "", err
	}
	if user.Role == nil {
		return 0, "", errors.New("角色不存在")
	}
	if !user.Role.IsActive {
		return 0, "", errors.New("职位已停用")
	}
	if user.Department == nil {
		return 0, "", errors.New("部门不存在")
	}
	if !user.Department.IsActive {
		return 0, "", errors.New("部门已停用")
	}
	return user.Role.Level, user.Department.Name, nil
}

func (u *UserRepository) GetDepartmentIDAndRoleIDByID(ctx context.Context, id uint64) (uint64, uint64, error) {
	var user model.User
	err := u.DB.WithContext(ctx).Preload("Department").Preload("Role").First(&user, id).Error
	if err != nil {
		return 0, 0, err
	}
	if user.Role == nil {
		return 0, 0, errors.New("职位不存在")
	}
	if !user.Role.IsActive {
		return 0, 0, errors.New("职位已停用")
	}
	if user.Department == nil {
		return 0, 0, errors.New("部门不存在")
	}
	if !user.Department.IsActive {
		return 0, 0, errors.New("部门已停用")
	}
	return user.Department.ID, user.Role.ID, nil
}

func (u *UserRepository) SaveRefreshToken(ctx context.Context, id uint64, device string, token string, times uint) error {
	key := fmt.Sprintf("%d-%s", id, device)
	err := u.Rdb.Set(ctx, key, token, time.Duration(times)*time.Hour*24).Err()
	if err != nil {
		return errors.New("存入redis失败，" + err.Error())
	}
	res := model.RefreshToken{
		Device: device,
		Token:  token,
		UserID: id,
	}
	err1 := u.DB.WithContext(ctx).Where(model.RefreshToken{UserID: id, Device: device}).
		Assign(model.RefreshToken{Token: token}).
		FirstOrCreate(&res).Error
	if err1 != nil {
		return errors.New("更新refresh表失败")
	}
	return nil
}

func (u *UserRepository) DeleteRefreshToken(ctx context.Context, id uint64, device string) error {
	key := fmt.Sprintf("%d-%s", id, device)
	err := u.Rdb.Del(ctx, key).Err()
	if err != nil {
		return errors.New("redis删除失败")
	}
	err1 := u.DB.WithContext(ctx).Where("user_id = ? AND device = ? ", id, device).Delete(&model.RefreshToken{}).Error
	if err1 != nil {
		return errors.New("refresh表删除失败")
	}
	return nil
}

func (u *UserRepository) GetRefreshToken(ctx context.Context, id uint64, device string) (string, error) {
	key := fmt.Sprintf("%d-%s", id, device)
	token, err := u.Rdb.Get(ctx, key).Result()
	if err == nil {
		return token, nil
	}
	if !errors.Is(err, redis.Nil) {
		return "", errors.New("获取refresh_token失败: " + err.Error())
	}

	// Redis 缓存未命中（例如 Redis 重启或键过期），回源查询 MySQL
	var record model.RefreshToken
	dbErr := u.DB.WithContext(ctx).Where("user_id = ? AND device = ?", id, device).First(&record).Error
	if dbErr != nil {
		if errors.Is(dbErr, gorm.ErrRecordNotFound) {
			return "", errors.New("refresh_token不存在或者过期")
		}
		return "", errors.New("获取refresh_token失败: " + dbErr.Error())
	}

	// 回填 Redis，保证下次读取走缓存
	_ = u.Rdb.Set(ctx, key, record.Token, 15*24*time.Hour).Err()

	return record.Token, nil
}

func (u *UserRepository) GetRefreshTokenByToken(ctx context.Context, token string) (*model.RefreshToken, error) {
	var record model.RefreshToken
	dbErr := u.DB.WithContext(ctx).Where("token = ?", token).First(&record).Error
	if dbErr != nil {
		if errors.Is(dbErr, gorm.ErrRecordNotFound) {
			return nil, errors.New("refresh_token不存在或者过期")
		}
		return nil, errors.New("获取refresh_token失败: " + dbErr.Error())
	}

	// 校验 Redis：检查此用户在当前设备上的最新 token 是否仍然是此 token（防单端多设备顶号或过期）
	key := fmt.Sprintf("%d-%s", record.UserID, record.Device)
	cachedToken, err := u.Rdb.Get(ctx, key).Result()
	if err == nil {
		if cachedToken != token {
			return nil, errors.New("账号已在其他设备登录或已失效")
		}
	} else if errors.Is(err, redis.Nil) {
		// Redis 缓存未命中（例如 Redis 重启或键过期），回填 Redis
		_ = u.Rdb.Set(ctx, key, record.Token, 15*24*time.Hour).Err()
	} else {
		return nil, errors.New("获取refresh_token失败: " + err.Error())
	}

	return &record, nil
}

func (u *UserRepository) GetUserList(ctx context.Context, keyword string, department string, role string, departmentID uint64, roleID uint64, grade uint, isGraduated *bool, page int, pageSize int, isAll *bool) ([]model.UserInfo, int64, error) {
	var userList []model.UserInfo
	var users []model.User
	var total int64

	query := u.DB.WithContext(ctx).Model(&model.User{}).
		Joins("LEFT JOIN departments ON departments.id = users.department_id").
		Joins("LEFT JOIN roles ON roles.id = users.role_id")
	if departmentID != 0 {
		query = query.Where("users.department_id = ?", departmentID)
	} else if department != "" {
		query = query.Where("departments.name LIKE ?", "%"+department+"%")
	}
	if roleID != 0 {
		query = query.Where("users.role_id = ?", roleID)
	} else if role != "" {
		query = query.Where("roles.name LIKE ?", "%"+role+"%")
	}
	if grade != 0 {
		query = query.Where("users.grade = ?", grade)
	}
	if isGraduated != nil {
		query = query.Where("users.is_graduated = ?", *isGraduated)
	}
	if keyword != "" {
		kw := "%" + keyword + "%"
		query = query.Where("(users.username LIKE ? OR users.name LIKE ? OR users.student_id LIKE ?)", kw, kw, kw)
	}

	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, errors.New("查询用户数量失败: " + err.Error())
	}

	dataQuery := query.Session(&gorm.Session{}).
		Select("users.*").
		Preload("Department").
		Preload("Role")

	if isAll == nil || !*isAll {
		offset := (page - 1) * pageSize
		dataQuery = dataQuery.Limit(pageSize).Offset(offset)
	}

	if err := dataQuery.Find(&users).Error; err != nil {
		return nil, 0, errors.New("查询数据失败")
	}

	userList = make([]model.UserInfo, 0, len(users))
	now := time.Now()
	for _, user := range users {
		userInfo := model.UserInfo{
			UserID:    user.ID,
			StudentID: user.StudentID,
			Username:  user.Username,
			Name:      user.Name,
			Avatar:      model.Avatar{URI: user.Avatar},
			Email:       user.Email,
			Grade:       user.Grade,
			IsGraduated: user.IsGraduated,
		}
		if user.Role != nil {
			userInfo.Role = user.Role.Name
			userInfo.RoleLevel = user.Role.Level
		}
		if user.Department != nil {
			userInfo.Department = user.Department.Name
		}
		if user.ScheduledDeleteAt != nil && now.Before(*user.ScheduledDeleteAt) {
			userInfo.ScheduledDeleteAt = user.ScheduledDeleteAt.In(time.FixedZone("Asia/Shanghai", 8*60*60)).Format("2006-01-02 15:04:05")
			userInfo.IsDeleting = true
		}
		userList = append(userList, userInfo)
	}
	return userList, total, nil
}

func (u *UserRepository) SaveVerificationCode(ctx context.Context, id uint64, code string, types string, expire time.Duration) error {
	key := fmt.Sprintf("%d-%sVerificationCode", id, types)
	err := u.Rdb.Set(ctx, key, code, expire).Err()
	if err != nil {
		return errors.New("存入redis失败," + err.Error())
	}
	return nil
}

func (u *UserRepository) GetVerificationCode(ctx context.Context, id uint64, types string) (string, error) {
	key := fmt.Sprintf("%d-%sVerificationCode", id, types)
	code, err := u.Rdb.Get(ctx, key).Result()
	if err != nil {
		return "", errors.New("获取验证码失败，" + err.Error())
	}
	return code, nil
}

func (u *UserRepository) DeleteVerificationCode(ctx context.Context, id uint64, types string) error {
	key := fmt.Sprintf("%d-%sVerificationCode", id, types)
	if err := u.Rdb.Del(ctx, key).Err(); err != nil {
		return errors.New("删除验证码失败，" + err.Error())
	}
	return nil
}

func (u *UserRepository) SetCooldown(ctx context.Context, id uint64, cooldown time.Duration) error {
	key := fmt.Sprintf("%d-CoolDown", id)
	err := u.Rdb.Set(ctx, key, "cooldown", cooldown).Err()
	if err != nil {
		return errors.New("设置失败" + err.Error())
	}
	return nil
}

func (u *UserRepository) SetCooldownNX(ctx context.Context, id uint64, token string, cooldown time.Duration) (bool, error) {
	key := fmt.Sprintf("%d-CoolDown", id)
	success, err := u.Rdb.SetNX(ctx, key, token, cooldown).Result()
	if err != nil {
		return false, errors.New("设置冷却时间失败: " + err.Error())
	}
	return success, nil
}

func (u *UserRepository) DeleteCooldown(ctx context.Context, token string, id uint64) error {
	key := fmt.Sprintf("%d-CoolDown", id)
	return releaseCooldownScript.Run(ctx, u.Rdb, []string{key}, token).Err()
}

func (u *UserRepository) CheckCooldown(ctx context.Context, id uint64) (bool, error) {
	key := fmt.Sprintf("%d-CoolDown", id)
	_, err := u.Rdb.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil
		}
		return false, errors.New("检查冷却时间失败: " + err.Error())
	}
	return true, nil
}

// 安全校验验证码
func (u *UserRepository) VerifyAndConsumeVerificationCode(ctx context.Context, id uint64, types string, inputCode string, maxFails int64) error {
	codeKey := fmt.Sprintf("%d-%sVerificationCode", id, types)
	failKey := fmt.Sprintf("%d-%sVerificationCode-Fails", id, types)
	// 1. 获取真实验证码
	expectedCode, err := u.Rdb.Get(ctx, codeKey).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return errors.New("验证码错误或已失效")
		}
		return errors.New("获取验证码失败: " + err.Error())
	}
	// 2. 恒定时间比对
	if subtle.ConstantTimeCompare([]byte(expectedCode), []byte(inputCode)) != 1 {
		// 校验失败，累计失败次数
		fails, incrErr := u.Rdb.Incr(ctx, failKey).Result()
		if incrErr == nil {
			if fails == 1 {
				// 获取验证码剩余的 TTL，让失败计数器与验证码同步过期
				if ttl, ttlErr := u.Rdb.TTL(ctx, codeKey).Result(); ttlErr == nil && ttl > 0 {
					_ = u.Rdb.Expire(ctx, failKey, ttl).Err()
				}
			}
			// 连续输错 5 次，立即销毁验证码
			if fails >= maxFails {
				_ = u.Rdb.Del(ctx, codeKey, failKey).Err()
				return errors.New("验证码错误次数过多，已作废，请重新获取")
			}
		}
		return errors.New("验证码错误")
	}
	// 3. 校验成功，销毁验证码与失败计数器
	_ = u.Rdb.Del(ctx, codeKey, failKey).Err()
	return nil
}

func (u *UserRepository) BatchUserInfoList(ctx context.Context, items []model.UpdateUserItems) error {
	if len(items) == 0 {
		return nil
	}
	return u.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, item := range items {
			updateData := make(map[string]any)
			if item.DepartmentID != nil {
				updateData["department_id"] = *item.DepartmentID
			}
			if item.RoleID != nil {
				updateData["role_id"] = *item.RoleID
			}
			if item.Grade != nil {
				updateData["grade"] = *item.Grade
			}
			if item.IsGraduated != nil {
				updateData["is_graduated"] = *item.IsGraduated
			}
			if len(updateData) > 0 {
				err := tx.Model(&model.User{}).
					Where("id = ? ", item.UserID).
					Updates(updateData).Error
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (u *UserRepository) GetUsersInfo(ctx context.Context, ids []uint64) (map[uint64]model.BatchUserInfo, error) {
	var userList []model.User
	err := u.DB.WithContext(ctx).Model(&model.User{}).
		Preload("Department").
		Preload("Role").
		Where("id in (?)", ids).
		Find(&userList).Error
	if err != nil {
		return nil, errors.New("批量获取数据失败")
	}
	var res = make(map[uint64]model.BatchUserInfo, len(ids))
	for _, user := range userList {
		temp := model.BatchUserInfo{
			StudentID: user.StudentID,
			Username:  user.Username,
			Name:      user.Name,
		}
		res[user.ID] = temp
	}
	return res, nil
}

func (u *UserRepository) GetUserMapByUserIDs(ctx context.Context, ids []uint64) (map[uint64]model.UserInfo, error) {
	var userList []model.User
	err := u.DB.WithContext(ctx).Model(&model.User{}).
		Preload("Department").
		Preload("Role").
		Where("id in (?)", ids).
		Find(&userList).Error
	if err != nil {
		return nil, err
	}

	res := make(map[uint64]model.UserInfo, len(userList))
	for _, user := range userList {
		departmentName := ""
		roleName := ""
		if user.Department != nil {
			departmentName = user.Department.Name
		}
		if user.Role != nil {
			roleName = user.Role.Name
		}
		res[user.ID] = model.UserInfo{
			UserID:     user.ID,
			StudentID:  user.StudentID,
			Username:   user.Username,
			Name:       user.Name,
			Department: departmentName,
			Role:       roleName,
		}
	}
	return res, nil
}

func (u *UserRepository) GetDepartmentByUserIDs(ctx context.Context, userIDs []uint64) (map[uint64]uint64, error) {
	result := make(map[uint64]uint64)
	var users []*model.User
	err := u.DB.WithContext(ctx).Where("id IN (?)", userIDs).Find(&users).Error
	if err != nil {
		return nil, err
	}
	for _, value := range users {
		result[value.ID] = value.DepartmentID
	}
	return result, nil
}

func (u *UserRepository) FindUsersByIDs(ctx context.Context, ids []uint64) ([]model.User, error) {
	if len(ids) == 0 {
		return []model.User{}, nil
	}
	var users []model.User
	err := u.DB.WithContext(ctx).Where("id IN (?)", ids).Find(&users).Error
	return users, err
}

func (u *UserRepository) CheckUserIsHave(ctx context.Context, id uint64) error {
	var count int64
	err := u.DB.WithContext(ctx).Model(&model.User{}).Where("id = ? ", id).Count(&count).Error
	if err != nil {
		return err
	}
	if count == 0 {
		return errors.New("用户不存在")
	}
	return nil
}

func (u *UserRepository) DeleteUserByID(ctx context.Context, id uint64) error {
	// 1. 查询该用户在各设备的 RefreshToken，并在 Redis 中同步清理
	var tokens []model.RefreshToken
	_ = u.DB.WithContext(ctx).Where("user_id = ?", id).Find(&tokens).Error
	for _, t := range tokens {
		key := fmt.Sprintf("%d-%s", id, t.Device)
		_ = u.Rdb.Del(ctx, key).Err()
	}

	// 2. 清理 Redis 中可能残留的各场景验证码与冷却时间
	_ = u.Rdb.Del(ctx,
		fmt.Sprintf("%d-reset_passwordVerificationCode", id),
		fmt.Sprintf("%d-delete_userVerificationCode", id),
		fmt.Sprintf("%d-cancel_deleteVerificationCode", id),
		fmt.Sprintf("%d-CoolDown", id),
	).Err()

	// 3. 事务删除该用户的 refresh_tokens 并软删除用户
	return u.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", id).Delete(&model.RefreshToken{}).Error; err != nil {
			return err
		}
		result := tx.Delete(&model.User{}, id)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errors.New("数据不存在")
		}
		return nil
	})
}

func (u *UserRepository) SetScheduledDeleteAt(ctx context.Context, id uint64, scheduledDeleteAt *time.Time) error {
	return u.DB.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Update("scheduled_delete_at", scheduledDeleteAt).Error
}

func (u *UserRepository) GetDueScheduledDeleteUsers(ctx context.Context, now time.Time) ([]model.User, error) {
	var users []model.User
	err := u.DB.WithContext(ctx).
		Where("scheduled_delete_at IS NOT NULL AND scheduled_delete_at <= ?", now).
		Find(&users).Error
	return users, err
}

func (u *UserRepository) GetDueGraduationUsers(ctx context.Context, maxGrade uint) ([]model.User, error) {
	var users []model.User
	err := u.DB.WithContext(ctx).
		Preload("Role").
		Where("is_graduated = ? AND grade > 0 AND grade <= ?", false, maxGrade).
		Find(&users).Error
	return users, err
}

func (u *UserRepository) GraduateUsers(ctx context.Context, nonDevIDs []uint64, devIDs []uint64, memberRoleID uint64, defaultDeptID uint64) error {
	if len(nonDevIDs) == 0 && len(devIDs) == 0 {
		return nil
	}
	return u.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if len(nonDevIDs) > 0 {
			if err := tx.Model(&model.User{}).
				Where("id IN (?) AND is_graduated = ?", nonDevIDs, false).
				Updates(map[string]any{
					"is_graduated":  true,
					"department_id": defaultDeptID,
					"role_id":       memberRoleID,
				}).Error; err != nil {
				return err
			}
		}
		if len(devIDs) > 0 {
			if err := tx.Model(&model.User{}).
				Where("id IN (?) AND is_graduated = ?", devIDs, false).
				Updates(map[string]any{
					"is_graduated": true,
				}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
