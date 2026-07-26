package service

import (
	"errors"
	"growthos/internal/model"
	"growthos/internal/repository"
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
