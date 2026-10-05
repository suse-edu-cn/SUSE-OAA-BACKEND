package repository

import (
	"context"
	"errors"
	"suseoaa/internal/model"

	"gorm.io/gorm"
)

type DepartmentRepository struct {
	DB *gorm.DB
}

func NewDepartmentRepository(db *gorm.DB) *DepartmentRepository {
	return &DepartmentRepository{
		DB: db,
	}
}

func (d *DepartmentRepository) FindByName(ctx context.Context, name string) (*model.Department, error) {
	var department model.Department
	err := d.DB.WithContext(ctx).Where("name = ?", name).First(&department).Error
	if err != nil {
		return nil, errors.New("查询部门失败" + err.Error())
	}
	return &department, nil
}

func (d *DepartmentRepository) FindAll(ctx context.Context) (*[]model.Department, error) {
	var departments []model.Department
	err := d.DB.WithContext(ctx).Find(&departments).Error
	if err != nil {
		return nil, errors.New("查询所有部门失败" + err.Error())
	}
	return &departments, nil
}

func (d *DepartmentRepository) GetDepartmentMap(ctx context.Context) (map[uint64]*model.Department, error) {
	result := make(map[uint64]*model.Department)
	departments, err := d.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	for _, value := range *departments {
		v := value
		result[v.ID] = &v
	}
	return result, nil
}

func (d *DepartmentRepository) GetDepartmentByID(ctx context.Context, id uint64) (model.Department, error) {
	var department model.Department
	err := d.DB.WithContext(ctx).Where("id = ?", id).First(&department).Error
	if err != nil {
		return model.Department{}, err
	}
	return department, nil
}

func (d *DepartmentRepository) GetDepartmentByName(ctx context.Context, name string) (*model.Department, error) {
	var department model.Department
	err := d.DB.WithContext(ctx).Where("name = ?", name).First(&department).Error
	if err != nil {
		return nil, err
	}
	return &department, nil
}

func (d *DepartmentRepository) CreateDepartment(ctx context.Context, department *model.Department) error {
	return d.DB.WithContext(ctx).Create(department).Error
}

func (d *DepartmentRepository) UpdateDepartment(ctx context.Context, department *model.Department, isActive *bool) error {
	updates := map[string]any{
		"name": department.Name,
		"type": department.Type,
	}
	if isActive != nil {
		updates["is_active"] = *isActive
	}

	var existing model.Department
	if err := d.DB.WithContext(ctx).Select("id").First(&existing, department.ID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("部门不存在")
		}
		return err
	}

	tx := d.DB.WithContext(ctx).Model(&model.Department{}).
		Where("id = ?", department.ID).
		Updates(updates)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (d *DepartmentRepository) GetTypeByDepartmentID(ctx context.Context, departmentID uint64) (string, error) {
	var department model.Department
	err := d.DB.WithContext(ctx).Where("id = ?", departmentID).First(&department).Error
	if err != nil {
		return "", err
	}
	return department.Type, nil
}

func (d *DepartmentRepository) GetDepartmentByType(ctx context.Context, departmentType string) ([]*model.Department, error) {
	var departments []*model.Department
	tx := d.DB.WithContext(ctx).Model(&model.Department{}).Where("is_active = ?", true)
	if departmentType != "" {
		tx = tx.Where("type = ?", departmentType)
	}
	err := tx.Find(&departments).Error
	if err != nil {
		return nil, err
	}
	return departments, nil
}
