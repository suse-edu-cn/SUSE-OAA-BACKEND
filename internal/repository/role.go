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
func (r *RoleRepository) UpdateRole(role *model.Role, isActive *bool) error {
	updates := map[string]any{
		"name":  role.Name,
		"type":  role.Type,
		"level": role.Level,
	}
	if isActive != nil {
		updates["is_active"] = *isActive
	}

	var existing model.Role
	if err := r.DB.Select("id").First(&existing, role.ID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("职位不存在")
		}
		return err
	}

	tx := r.DB.Model(&model.Role{}).
		Where("id = ?", role.ID).
		Updates(updates)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
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

func (r *RoleRepository) GetActiveRoleByUserID(id uint64) (uint64, uint64, error) {
	var user model.User
	err := r.DB.Preload("Department").Preload("Role").Where("id = ?", id).First(&user).Error
	if err != nil {
		return 0, 0, err
	}
	if user.Role == nil {
		return 0, 0, errors.New("用户role缺失")
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
	return user.Role.ID, user.Role.Level, nil
}
func (r *RoleRepository) GetTypeByRoleID(roleID uint64) (string, error) {
	var role model.Role
	err := r.DB.Model(&model.Role{}).Where("id = ?", roleID).First(&role).Error
	if err != nil {
		return "", err
	}
	return role.Type, nil
}
func (r *RoleRepository) GetRoleByType(roleType string) ([]*model.Role, error) {
	var roles []*model.Role
	tx := r.DB.Model(&model.Role{}).Where("is_active = ?", true)
	if roleType != "" {
		tx = tx.Where("type = ?", roleType)
	}
	err := tx.Find(&roles).Error
	if err != nil {
		return nil, err
	}
	return roles, nil
}
func (r *RoleRepository) GetRoleByID(roleID uint64) (*model.Role, error) {
	var role model.Role
	err := r.DB.Model(&model.Role{}).Where("id = ?", roleID).First(&role).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}
