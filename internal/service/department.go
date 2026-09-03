package service

import (
	"errors"
	"suseoaa/internal/model"
	"suseoaa/internal/repository"
)

type DepartmentService struct {
	DepartmentRepo repository.DepartmentRepository
	RoleRepo       repository.RoleRepository
}

func NewDepartmentService(departmentRepo repository.DepartmentRepository,
	roleRepo repository.RoleRepository) DepartmentService {
	return DepartmentService{
		DepartmentRepo: departmentRepo,
		RoleRepo:       roleRepo,
	}
}

func (d *DepartmentService) GetAll() (*[]model.Department, error) {
	departments, err := d.DepartmentRepo.FindAll()
	if err != nil {
		return nil, errors.New("获取失败" + err.Error())
	}
	return departments, nil
}

func (d *DepartmentService) CreateDepartment(id uint64, department *model.Department) error {
	_, level, err := d.RoleRepo.GetActiveRoleByUserID(id)
	if err != nil {
		return err
	}
	if level < 80 {
		return errors.New("权限不够")
	}
	return d.DepartmentRepo.CreateDepartment(department)
}
func (d *DepartmentService) UpdateDepartment(id uint64, department *model.Department, isActive *bool) error {
	_, level, err := d.RoleRepo.GetActiveRoleByUserID(id)
	if err != nil {
		return err
	}
	if level < 80 {
		return errors.New("权限不够")
	}
	if _, err := d.DepartmentRepo.GetDepartmentByID(department.ID); err != nil {
		return err
	}
	return d.DepartmentRepo.UpdateDepartment(department, isActive)
}
