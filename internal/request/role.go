package request

import "errors"

type CreateRoleReq struct {
	Name  string `json:"name" binding:"required"`
	Level uint64 `json:"level" binding:"required"`
	Type  string `json:"type" binding:"required"`
}
type UpdateRoleReq struct {
	RoleID uint64 `json:"role_id" binding:"required"`
	Name   string `json:"name" binding:"required"`
	Level  uint64 `json:"level" binding:"required"`
	Type   string `json:"type" binding:"required"`
}

func (c *CreateRoleReq) CheckType() error {
	if c.Type == "开发" || c.Type == "普通" || c.Type == "部门" || c.Type == "协会" {
		return nil
	}
	return errors.New("type 错误")
}

func (c *UpdateRoleReq) CheckType() error {
	if c.Type == "开发" || c.Type == "普通" || c.Type == "部门" || c.Type == "协会" {
		return nil
	}
	return errors.New("type 错误")
}
