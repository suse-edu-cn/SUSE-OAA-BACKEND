package repository

import (
	"errors"
	"suseoaa/internal/model"

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
		v := value
		result[v.ID] = &v
	}
	return result, nil
}
func (d *DepartmentRepository) GetDepartmentByID(id uint64) (model.Department, error) {
	var department model.Department
	err := d.DB.Where("id = ?", id).First(&department).Error
	if err != nil {
		return model.Department{}, err
	}
	return department, nil
}
func (d *DepartmentRepository) GetDepartmentByName(name string) (*model.Department, error) {
	var department model.Department
	err := d.DB.Where("name = ?", name).First(&department).Error
	if err != nil {
		return nil, err
	}
	return &department, nil
}

func (d *DepartmentRepository) CreateDepartment(department *model.Department) error {
	return d.DB.Create(department).Error
}
func (d *DepartmentRepository) UpdateDepartment(department *model.Department) error {
	return d.DB.Model(&model.Department{}).Where("id = ?", department.ID).Updates(department).Error
}

func (d *DepartmentRepository) GetTypeByDepartmentID(departmentID uint64) (string, error) {
	var department model.Department
	err := d.DB.Where("id = ?", departmentID).First(&department).Error
	if err != nil {
		return "", err
	}
	return department.Type, nil
}
func (d *DepartmentRepository) GetDepartmentByType(departmentType string) ([]*model.Department, error) {
	var departments []*model.Department
	tx := d.DB.Model(&model.Department{}).Where("is_active = ?", true)
	if departmentType != "" {
		tx = tx.Where("type = ?", departmentType)
	}
	err := tx.Find(&departments).Error
	if err != nil {
		return nil, err
	}
	return departments, nil
}

//func (d *DepartmentRepository) GetDepartmentByID(id uint64) error {}
