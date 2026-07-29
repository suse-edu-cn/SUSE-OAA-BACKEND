package repository

import (
	"context"
	"errors"
	"fmt"
	"growthos/internal/model"
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
func (u *UserRepository) CheckExist(studentId string, email string) error {
	var num int64
	err := u.DB.Model(&model.User{}).Where("email = ? or student_id = ? ", email, studentId).Count(&num).Error
	if err != nil {
		return errors.New("查询失败，" + err.Error())
	}
	if num > 0 {
		return errors.New("学号或者邮箱已存在")
	}
	return nil
}

func (u *UserRepository) FindUserById(id uint64) (model.User, error) {
	var user model.User
	err := u.DB.Where("id = ?", id).First(&user).Error
	return user, err
}

func (u *UserRepository) UpdateUserInfo(user model.User) error {
	return u.DB.Save(&user).Error
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
	info.Email = user.Email
	info.Username = user.Username
	info.Name = user.Name
	info.StudentID = user.StudentID

	return info, nil
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
	fmt.Println(token)
	fmt.Println(key)
	fmt.Println(err)
	if errors.Is(err, redis.Nil) {
		return "", errors.New("refresh_token不存在或者过期")
	}
	return token, nil
}
