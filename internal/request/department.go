package request

type CreateDepartmentReq struct {
	Name string `json:"name" binding:"required"`
}
type UpdateDepartmentReq struct {
	DepartmentID uint64 `json:"department_id" binding:"required"`
	Name         string `json:"name" binding:"required"`
}
