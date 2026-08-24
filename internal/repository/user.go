package repository

import (
	"context"
	"errors"
	"fmt"
	"suseoaa/internal/model"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type UserRepository struct {
	DB  *gorm.DB
	Rdb *redis.Client
}

func NewUserRepository(db *gorm.DB, rdb *redis.Client) UserRepository {
	return UserRepository{
		DB:  db,
		Rdb: rdb,
	}
}

func (u *UserRepository) CreateUser(user model.User) error {
	return u.DB.Create(&user).Error
}

func (u *UserRepository) FindUserByAccount(message string) (model.User, error) {
	var user model.User
	err := u.DB.Model(&model.User{}).Where("username = ? or email = ? or student_id = ? ", message, message, message).First(&user).Error
	return user, err
}

func (u *UserRepository) FindUserByUsername(username string) (model.User, error) {
	var user model.User
	err := u.DB.Model(&model.User{}).Where("username = ?", username).First(&user).Error
	return user, err
}
func (u *UserRepository) CheckExist(studentId string, email string, username string) error {
	var num int64
	err := u.DB.Model(&model.User{}).
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

func (u *UserRepository) FindUserById(id uint64) (model.User, error) {
	var user model.User
	err := u.DB.Where("id = ?", id).First(&user).Error
	return user, err
}
func (u *UserRepository) ResetPassword(id uint64, password string) error {
	err := u.DB.Model(&model.User{}).Where("id = ?", id).Update("password", password).Error
	if err != nil {
		return errors.New("更新失败")
	}
	return nil
}

func (u *UserRepository) UpdateUsername(id uint64, username string) error {
	err := u.DB.Model(&model.User{}).Where("id = ?", id).Update("username", username).Error
	if err != nil {
		return errors.New("更新失败")
	}
	return nil
}

func (u *UserRepository) GetUserInfoById(id uint64) (model.UserInfo, error) {
	var user model.User
	var info model.UserInfo
	err := u.DB.Preload("Department").Preload("Role").First(&user, id).Error
	if err != nil {
		return info, err
	}
	if user.Role != nil {
		info.Role = user.Role.Name
	}
	if user.Department != nil {
		info.Department = user.Department.Name
	}
	info.ID = id
	info.Email = user.Email
	info.Username = user.Username
	info.Name = user.Name
	info.StudentID = user.StudentID

	return info, nil
}
func (u *UserRepository) GetRoleLevelAndDepartment(id uint64) (uint64, string, error) {
	var user model.User
	err := u.DB.Preload("Department").Preload("Role").First(&user, id).Error
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

func (u *UserRepository) SaveRefreshToken(id uint64, device string, token string, times uint, ctx context.Context) error {
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
	err1 := u.DB.Where(model.RefreshToken{UserID: id, Device: device}).
		Assign(model.RefreshToken{Token: token}).
		FirstOrCreate(&res).Error
	if err1 != nil {
		return errors.New("更新refresh表失败")
	}
	return nil
}

func (u *UserRepository) DeleteRefreshToken(id uint64, device string, ctx context.Context) error {
	key := fmt.Sprintf("%d-%s", id, device)
	err := u.Rdb.Del(ctx, key).Err()
	if err != nil {
		return errors.New("redis删除失败")
	}
	err1 := u.DB.Where("user_id = ? AND device = ? ", id, device).Delete(&model.RefreshToken{}).Error
	if err1 != nil {
		return errors.New("refresh表删除失败")
	}
	return nil
}
func (u *UserRepository) GetRefreshToken(id uint64, device string, ctx context.Context) (string, error) {
	key := fmt.Sprintf("%d-%s", id, device)
	token, err := u.Rdb.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", errors.New("refresh_token不存在或者过期")
	}
	return token, nil
}
func (u *UserRepository) GetUserList(keyword string, department string, role string, page int, pageSize int) ([]model.UserInfo, int64, error) {
	var userList []model.UserInfo
	var users []model.User
	var total int64
	query := u.DB.Model(&model.User{}).
		Joins("LEFT JOIN departments ON departments.id = users.department_id").
		Joins("LEFT JOIN roles ON roles.id = users.role_id")

	if department != "" {
		query = query.Where("departments.name LIKE ?", "%"+department+"%")
	}
	if role != "" {
		query = query.Where("roles.name LIKE ?", "%"+role+"%")
	}
	if keyword != "" {
		kw := "%" + keyword + "%"
		query = query.Where("(users.username LIKE ? OR users.name LIKE ? OR users.student_id LIKE ?)", kw, kw, kw)
	}
	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, errors.New("查询用户数量失败: " + err.Error())
	}
	offset := (page - 1) * pageSize
	err = query.Select("users.*").
		Preload("Department").
		Preload("Role").
		Limit(pageSize).
		Offset(offset).
		Find(&users).Error
	if err != nil {
		return nil, 0, errors.New("查询数据失败")
	}

	for _, user := range users {
		userInfo := model.UserInfo{
			ID:        user.ID,
			StudentID: user.StudentID,
			Username:  user.Username,
			Name:      user.Name,
			Email:     user.Email,
		}
		if user.Role != nil {
			userInfo.Role = user.Role.Name
		}
		if user.Department != nil {
			userInfo.Department = user.Department.Name
		}
		userList = append(userList, userInfo)
	}
	return userList, total, nil
}

func (u *UserRepository) SaveVerificationCode(id uint64, code string, types string, expire time.Duration, ctx context.Context) error {
	key := fmt.Sprintf("%d-%sVerificationCode", id, types)
	err := u.Rdb.Set(ctx, key, code, expire).Err()
	if err != nil {
		return errors.New("存入redis失败," + err.Error())
	}
	return nil
}
func (u *UserRepository) GetVerificationCode(id uint64, types string, ctx context.Context) (string, error) {
	key := fmt.Sprintf("%d-%sVerificationCode", id, types)
	code, err := u.Rdb.Get(ctx, key).Result()
	if err != nil {
		return "", errors.New("获取验证码失败，" + err.Error())
	}
	return code, nil
}
func (u *UserRepository) SetCooldown(id uint64, cooldown time.Duration, ctx context.Context) error {
	key := fmt.Sprintf("%d-CoolDown", id)
	err := u.Rdb.Set(ctx, key, "cooldown", cooldown).Err()
	if err != nil {
		return errors.New("设置失败" + err.Error())
	}
	return nil
}

func (u *UserRepository) CheckCooldown(id uint64, ctx context.Context) bool {
	key := fmt.Sprintf("%d-CoolDown", id)
	_, err := u.Rdb.Get(ctx, key).Result()
	if err != nil {
		return false
	}
	return true
}

func (u *UserRepository) BatchUserInfoList(ctx context.Context, items []model.UpdateUserItems) error {
	if len(items) == 0 {
		return nil
	}
	return u.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, item := range items {
			updateData := map[string]any{
				"department_id": item.DepartmentID,
				"role_id":       item.RoleID,
			}
			err := tx.Model(&model.User{}).
				Where("id = ? ", item.UserID).
				Updates(updateData).Error
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (u *UserRepository) GetUsersInfo(ids []uint64) (map[uint64]model.BatchUserInfo, error) {
	var userList []model.User
	err := u.DB.Model(&model.User{}).
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
		if user.Department != nil {
			temp.Department = user.Department.Name
		}
		if user.Role != nil {
			temp.Role = user.Role.Name
		}
		res[user.ID] = temp
	}
	return res, nil
}
