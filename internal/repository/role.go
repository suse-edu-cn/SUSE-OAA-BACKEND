package repository

import (
	"errors"
	"suseoaa/internal/model"

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
		v := value
		res[v.ID] = &v
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

func (r *RoleRepository) GetLevelByID(id uint64) (uint64, error) {
	var role model.Role
	err := r.DB.Model(&model.Role{}).Where("id = ?", id).First(&role).Error
	if err != nil {
		return 0, errors.New("获取level失败" + err.Error())
	}
	return role.Level, nil
}

func (r *RoleRepository) CreateRole(role *model.Role) error {
	return r.DB.Create(role).Error
}
func (r *RoleRepository) UpdateRole(role *model.Role) error {
	return r.DB.Model(&model.Role{}).Where("id = ?", role.ID).Updates(role).Error
}

func (r *RoleRepository) GetRoleByUserID(id uint64) (uint64, uint64, error) {
	var user model.User
	err := r.DB.Preload("Role").Where("id = ?", id).First(&user).Error
	if err != nil {
		return 0, 0, err
	}
	if user.Role != nil {
		return user.Role.ID, user.Role.Level, nil
	}
	return 0, 0, errors.New("用户role缺失")
}
