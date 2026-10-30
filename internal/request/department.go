package request

import "errors"

type CreateDepartmentReq struct {
	Name string `json:"name" binding:"required"`
	Type string `json:"type" binding:"required"`
}
type UpdateDepartmentReq struct {
	DepartmentID uint64 `json:"department_id" binding:"required"`
	Name         string `json:"name" binding:"required"`
	Type         string `json:"type" binding:"required"`
	IsActive     *bool  `json:"is_active" binding:"required"`
}

func (c CreateDepartmentReq) CheckType() error {
	if c.Type == "部门" || c.Type == "协会" || c.Type == "理事会" {
		return nil
	}
	return errors.New("类型不合法，必须为：部门、协会、理事会 之一")
}
func (d UpdateDepartmentReq) CheckType() error {
	if d.Type == "部门" || d.Type == "协会" || d.Type == "理事会" {
		return nil
	}
	return errors.New("类型不合法，必须为：部门、协会、理事会 之一")
}
