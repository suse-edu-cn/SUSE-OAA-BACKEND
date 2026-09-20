package handler

import (
	"suseoaa/internal/model"
	"suseoaa/internal/request"
	"suseoaa/internal/service"
	"suseoaa/pkg/response"

	"github.com/gin-gonic/gin"
)

type AnnouncementHandler struct {
	AnnouncementService service.AnnouncementService
}

func NewAnnouncementHandler(announcementService service.AnnouncementService) AnnouncementHandler {
	return AnnouncementHandler{
		AnnouncementService: announcementService,
	}
}
func (a *AnnouncementHandler) CreateAnnouncement(c *gin.Context) {
	var req request.CreateAnnouncementReq
	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.Fail(c, 400, "获取参数失败", nil)
		return
	}
	userID := c.GetUint64("user_id")
	announcement := model.Announcement{
		Title:        req.Title,
		CreatedId:    userID,
		Content:      req.Content,
		DepartmentID: req.DepartmentID,
	}

	res, err := a.AnnouncementService.CreateAnnouncement(userID, announcement)
	if err != nil {
		response.Fail(c, 500, err.Error(), nil)
		return
	}
	response.Success(c, res)
	return
}

func (a *AnnouncementHandler) UpdateAnnouncement(c *gin.Context) {
	var req request.UpdateAnnouncementReq
	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.Fail(c, 400, "获取参数失败", nil)
		return
	}
	userID := c.GetUint64("user_id")
	announcement := model.Announcement{
		ID:      req.AnnouncementID,
		Title:   req.Title,
		Content: req.Content,
	}
	err = a.AnnouncementService.UpdateAnnouncement(userID, announcement)
	if err != nil {
		response.Fail(c, 400, err.Error(), nil)
		return
	}
	response.Success(c, "公告更新成功")
	return
}

func (a *AnnouncementHandler) PushAnnouncement(c *gin.Context) {
	var req request.PushAnnouncementReq
	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.Fail(c, 400, "获取参数失败", nil)
		return
	}
	userID := c.GetUint64("user_id")
	ctx := c.Request.Context()
	err = a.AnnouncementService.PushAnnouncement(ctx, req.AnnouncementID, userID)
	if err != nil {
		response.Fail(c, 400, err.Error(), nil)
		return
	}
	response.Success(c, "推送成功")
	return
}

func (a *AnnouncementHandler) GetAnnouncementList(c *gin.Context) {
	var req request.GetAnnouncementListReq
	err := c.ShouldBindQuery(&req)
	if err != nil {
		response.Fail(c, 400, err.Error(), nil)
		return
	}
	id := c.GetUint64("user_id")
	ctx := c.Request.Context()
	isContent := false
	if req.Content != nil && *req.Content {
		isContent = true
	}
	announcementInfos, err := a.AnnouncementService.GetAnnouncementInfoList(ctx, id, req.Status, isContent)
	if err != nil {
		response.Fail(c, 400, err.Error(), nil)
		return
	}
	response.Success(c, announcementInfos)
	return
}
func (a *AnnouncementHandler) GetAnnouncement(c *gin.Context) {
	var req request.GetAnnouncementReq
	err := c.ShouldBindQuery(&req)
	if err != nil {
		response.Fail(c, 400, err.Error(), nil)
		return
	}
	userID := c.GetUint64("user_id")
	ctx := c.Request.Context()
	announcement, err := a.AnnouncementService.GetAnnouncementInfo(ctx, userID, req.AnnouncementID)
	if err != nil {
		response.Fail(c, 400, err.Error(), nil)
		return
	}
	response.Success(c, announcement)
	return

}
func (a *AnnouncementHandler) DeleteAnnouncement(c *gin.Context) {
	userID := c.GetUint64("user_id")
	var req request.DeleteAnnouncementReq
	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.Fail(c, 400, "获取参数失败", nil)
		return
	}
	err = a.AnnouncementService.DeleteAnnouncement(req.AnnouncementID, userID)
	if err != nil {
		response.Fail(c, 400, err.Error(), nil)
		return
	}
	response.Success(c, "删除公告成功")
	return
}
