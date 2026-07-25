package repository

import (
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
	err := u.DB.Where("username = ? or email = ? or student_id = ? ", message).First(&user).Error
	return user, err
}

func (u *UserRepository) FindUserByUsername(username string) (model.User, error) {
	var user model.User
	err := u.DB.Where("username = ?", username).First(&user).Error
	return user, err
}

func (u *UserRepository) FindUserById(id uint64) (model.User, error) {
	var user model.User
	err := u.DB.Where("id = ?", id).First(&user).Error
	return user, err
}

func (u *UserRepository) UpdateUserInfo(user model.User) error {
	return u.DB.Save(&user).Error
}
