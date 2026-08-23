package service

import (
	"errors"
	"suseoaa/internal/model"
	"suseoaa/internal/repository"
)

type DepartmentService struct {
	DepartmentRepo repository.DepartmentRepository
}

func NewDepartmentService(departmentRepo repository.DepartmentRepository) DepartmentService {
	return DepartmentService{
		DepartmentRepo: departmentRepo,
	}
}

func (d *DepartmentService) GetAll() (*[]model.Department, error) {
	departments, err := d.DepartmentRepo.FindAll()
	if err != nil {
		return nil, errors.New("获取失败" + err.Error())
	}
	return departments, nil
}
