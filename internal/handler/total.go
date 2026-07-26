package handler

type TotalHandler struct {
	Auth       AuthHandler
	User       UserHandler
	Department DepartmentHandler
	Role       RoleHandler
}

func NewTotalHandler(auth AuthHandler, user UserHandler, department DepartmentHandler, role RoleHandler) TotalHandler {
	return TotalHandler{
		Auth:       auth,
		User:       user,
		Department: department,
		Role:       role,
	}
}
