package request

type CreateRoleReq struct {
	Name  string `json:"name" binding:"required"`
	Level uint64 `json:"level" binding:"required"`
}
type UpdateRoleReq struct {
	RoleID uint64 `json:"role_id" binding:"required"`
	Name   string `json:"name" binding:"required"`
	Level  uint64 `json:"level" binding:"required"`
}
