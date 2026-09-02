package request

type UpdateUserInfoReq struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required"`
	Avatar   string `json:"avatar" binding:"required"`
}

type UserListReq struct {
	Keyword    string `form:"keyword"`
	Department string `form:"department"`
	Role       string `form:"role"`
	Page       int    `form:"page,default=1"`
	PageSize   int    `form:"page_size, default=20"`
}

type BatchUserInfoReq struct {
	UserID       uint64 `json:"user_id" binding:"required"`
	DepartmentID uint64 `json:"department_id" binding:"required"`
	RoleID       uint64 `json:"role_id" binding:"required"`
}
