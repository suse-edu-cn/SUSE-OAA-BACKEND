package handler

type TotalHandler struct {
	Auth         AuthHandler
	User         UserHandler
	Department   DepartmentHandler
	Role         RoleHandler
	Announcement AnnouncementHandler
	Term         TermHandler
	File         FileHandler
}

func NewTotalHandler(auth AuthHandler,
	user UserHandler,
	department DepartmentHandler,
	role RoleHandler,
	announcement AnnouncementHandler,
	term TermHandler,
	file FileHandler,
) TotalHandler {
	return TotalHandler{
		Auth:         auth,
		User:         user,
		Department:   department,
		Role:         role,
		Announcement: announcement,
		Term:         term,
		File:         file,
	}
}
