package request

type CreateAnnouncementReq struct {
	DepartmentID uint64 `json:"department_id" binding:"required"`
	Title        string `json:"title" binding:"required"`
	Content      string `json:"content" binding:"required"`
	IsPushed     *bool  `json:"is_pushed" binding:"required"`
}

type UpdateAnnouncementReq struct {
	AnnouncementID uint64 `json:"announcement_id" binding:"required"`
	Title          string `json:"title" binding:"required"`
	Content        string `json:"content" binding:"required"`
	IsPushed       *bool  `json:"is_pushed" binding:"required"`
}
type PushAnnouncementReq struct {
	AnnouncementID uint64 `json:"announcement_id" binding:"required"`
}
type DeleteAnnouncementReq struct {
	AnnouncementID uint64 `json:"announcement_id" binding:"required"`
}
