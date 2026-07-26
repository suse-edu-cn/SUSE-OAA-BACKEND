package repository

import (
	"errors"
	"growthos/internal/model"

	"gorm.io/gorm"
)

type RoleRepository struct {
	DB *gorm.DB
}

func NewRoleRepository(db *gorm.DB) RoleRepository {
	return RoleRepository{
		DB: db,
	}
}

func (r *RoleRepository) FindByName(name string) (*model.Role, error) {
	var role model.Role
	err := r.DB.Where("name = ?", name).First(&role).Error
	if err != nil {
		return nil, errors.New("查询职位失败" + err.Error())
	}
	return &role, nil
}
func (r *RoleRepository) FindAll() (*[]model.Role, error) {
	var roles []model.Role
	err := r.DB.Find(&roles).Error
	if err != nil {
		return nil, errors.New("查询所有职位失败" + err.Error())
	}
	return &roles, nil
}
