package repository

import (
	"errors"
	"growthos/internal/model"

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
