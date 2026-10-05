package service

import (
	"context"
	"errors"
	"suseoaa/internal/model"
	"suseoaa/internal/repository"
)

type RoleService struct {
	RoleRepo *repository.RoleRepository
}

func NewRoleService(roleRepo *repository.RoleRepository) *RoleService {
	return &RoleService{
		RoleRepo: roleRepo,
	}
}

func (r *RoleService) GetAll(ctx context.Context) (*[]model.Role, error) {
	role, err := r.RoleRepo.FindAll(ctx)
	if err != nil {
		return nil, errors.New("查询失败" + err.Error())
	}
	return role, nil
}

func (r *RoleService) GetRoleByUserID(ctx context.Context, id uint64) (uint64, uint64, error) {
	return r.RoleRepo.GetRoleByUserID(ctx, id)
}

func (r *RoleService) GetActiveRoleByUserID(ctx context.Context, id uint64) (uint64, uint64, error) {
	return r.RoleRepo.GetActiveRoleByUserID(ctx, id)
}

func (r *RoleService) Create(ctx context.Context, id uint64, role *model.Role) error {
	_, roleLevel, err := r.GetActiveRoleByUserID(ctx, id)
	if err != nil {
		return err
	}
	if roleLevel >= 80 {
		err = r.RoleRepo.CreateRole(ctx, role)
		if err != nil {
			return err
		}
		return nil
	}
	return errors.New("权限不够")
}

func (r *RoleService) Update(ctx context.Context, id uint64, role *model.Role, isActive *bool) error {
	_, operatorLevel, err := r.GetActiveRoleByUserID(ctx, id)
	if err != nil {
		return err
	}
	if operatorLevel < 80 {
		return errors.New("权限不够")
	}

	oldRole, err := r.RoleRepo.GetRoleByID(ctx, role.ID)
	if err != nil {
		return err
	}
	if oldRole.Level >= operatorLevel {
		return errors.New("不能修改同级或更高级职位")
	}
	if role.Level >= operatorLevel {
		return errors.New("不能把目标职位改到同级或更高级")
	}

	return r.RoleRepo.UpdateRole(ctx, role, isActive)
}
