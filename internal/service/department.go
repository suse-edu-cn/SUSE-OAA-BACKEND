package service

import (
	"context"
	"errors"
	"suseoaa/internal/model"
	"suseoaa/internal/repository"
)

type DepartmentService struct {
	DepartmentRepo *repository.DepartmentRepository
	RoleRepo       *repository.RoleRepository
}

func NewDepartmentService(departmentRepo *repository.DepartmentRepository,
	roleRepo *repository.RoleRepository) *DepartmentService {
	return &DepartmentService{
		DepartmentRepo: departmentRepo,
		RoleRepo:       roleRepo,
	}
}

func (d *DepartmentService) GetAll(ctx context.Context) (*[]model.Department, error) {
	departments, err := d.DepartmentRepo.FindAll(ctx)
	if err != nil {
		return nil, errors.New("获取失败" + err.Error())
	}
	return departments, nil
}

func (d *DepartmentService) CreateDepartment(ctx context.Context, id uint64, department *model.Department) error {
	_, level, err := d.RoleRepo.GetActiveRoleByUserID(ctx, id)
	if err != nil {
		return err
	}
	if level < 80 {
		return errors.New("权限不够")
	}
	return d.DepartmentRepo.CreateDepartment(ctx, department)
}

func (d *DepartmentService) UpdateDepartment(ctx context.Context, id uint64, department *model.Department, isActive *bool) error {
	_, level, err := d.RoleRepo.GetActiveRoleByUserID(ctx, id)
	if err != nil {
		return err
	}
	if level < 80 {
		return errors.New("权限不够")
	}
	if _, err := d.DepartmentRepo.GetDepartmentByID(ctx, department.ID); err != nil {
		return err
	}
	return d.DepartmentRepo.UpdateDepartment(ctx, department, isActive)
}
