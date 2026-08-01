package repository

import (
	"errors"
	"growthos/internal/model"

	"gorm.io/gorm"
)

type DepartmentRepository struct {
	DB *gorm.DB
}

func NewDepartmentRepository(db *gorm.DB) DepartmentRepository {
	return DepartmentRepository{
		DB: db,
	}
}

func (d *DepartmentRepository) FindByName(name string) (*model.Department, error) {
	var department model.Department
	err := d.DB.Where("name = ?", name).First(&department).Error
	if err != nil {
		return nil, errors.New("查询部门失败" + err.Error())
	}
	return &department, nil
}
func (d *DepartmentRepository) FindAll() (*[]model.Department, error) {
	var departments []model.Department
	err := d.DB.Find(&departments).Error
	if err != nil {
		return nil, errors.New("查询所有部门失败" + err.Error())
	}
	return &departments, nil
}

func (d *DepartmentRepository) GetDepartmentMap() (map[uint64]*model.Department, error) {
	result := make(map[uint64]*model.Department)
	departments, err := d.FindAll()
	if err != nil {
		return nil, err
	}
	for _, value := range *departments {
		result[value.ID] = &value
	}
	return result, nil
}
