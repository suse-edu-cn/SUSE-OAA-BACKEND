package service

import (
	"errors"
	"suseoaa/internal/model"
	"suseoaa/internal/repository"
)

type RoleService struct {
	RoleRepo repository.RoleRepository
}

func NewRoleService(roleRepo repository.RoleRepository) RoleService {
	return RoleService{
		RoleRepo: roleRepo,
	}
}

func (r *RoleService) GetAll() (*[]model.Role, error) {
	role, err := r.RoleRepo.FindAll()
	if err != nil {
		return nil, errors.New("查询失败" + err.Error())
	}
	return role, nil
}
func (r *RoleService) GetRoleByUserID(id uint64) (uint64, uint64, error) {
	var user model.User
	err := r.RoleRepo.DB.Preload("Role").Where("id = ?", id).First(&user).Error
	if err != nil {
		return 0, 0, err
	}
	if user.Role != nil {
		return user.Role.ID, user.Role.Level, nil
	}
	return 0, 0, errors.New("用户role缺失")
}

func (r *RoleService) Create(id uint64, role *model.Role) error {
	_, roleLevel, err := r.GetRoleByUserID(id)
	if err != nil {
		return err
	}
	if roleLevel >= 80 {
		err = r.RoleRepo.CreateRole(role)
		if err != nil {
			return err
		}
		return nil
	}
	return errors.New("权限不够")
}
func (r *RoleService) Update(id uint64, role *model.Role) error {
	_, roleLevel, err := r.GetRoleByUserID(id)
	if err != nil {
		return err
	}
	if roleLevel >= 80 && role.Level < roleLevel {
		err = r.RoleRepo.UpdateRole(role)
		if err != nil {
			return err
		}
		return nil
	}
	return errors.New("权限不够")
}
