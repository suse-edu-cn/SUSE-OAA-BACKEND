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
func (r *RoleRepository) GetRoleMap() (map[uint64]*model.Role, error) {
	res := make(map[uint64]*model.Role)
	roles, err := r.FindAll()
	if err != nil {
		return nil, err
	}
	for _, value := range *roles {
		res[value.ID] = &value
	}
	return res, nil
}

func (r *RoleRepository) GetLevelByName(name string) (uint64, error) {
	var role model.Role
	err := r.DB.Model(&model.Role{}).Where("name = ?", name).First(&role).Error
	if err != nil {
		return 0, errors.New("获取" + name + "失败" + err.Error())
	}
	return role.Level, nil
}
