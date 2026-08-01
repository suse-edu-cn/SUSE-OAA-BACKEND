package request

type UpdateUserInfoReq struct {
	Username string `json:"username"`
}

type UserListReq struct {
	Keyword    string `form:"keyword"`
	Department string `form:"department"`
	Role       string `form:"role"`
	Page       int    `form:"page, default:1"`
	PageSize   int    `form:"page_size, default=20"`
}

type BatchUserInfoReq struct {
	UserID       uint64 `json:"user_id"`
	DepartmentID uint64 `json:"department_id"`
	RoleID       uint64 `json:"role_id"`
}
