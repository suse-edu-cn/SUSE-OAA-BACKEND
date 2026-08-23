package handler

type TotalHandler struct {
	Auth         AuthHandler
	User         UserHandler
	Department   DepartmentHandler
	Role         RoleHandler
	Announcement AnnouncementHandler
}

func NewTotalHandler(auth AuthHandler, user UserHandler, department DepartmentHandler, role RoleHandler,
	announcement AnnouncementHandler) TotalHandler {
	return TotalHandler{
		Auth:         auth,
		User:         user,
		Department:   department,
		Role:         role,
		Announcement: announcement,
	}
}
