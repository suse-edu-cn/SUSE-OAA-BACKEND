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
		response.Fail(c, 400, "获取参数失败")
		return
	}
	userID := c.GetUint64("user_id")
	announcement := model.Announcement{
		Title:        req.Title,
		CreatedId:    userID,
		Content:      req.Content,
		DepartmentID: req.DepartmentID,
	}

	_, err = a.AnnouncementService.CreateAnnouncement(userID, announcement)
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}
	response.Success(c, "公告创建成功")
	return
}

func (a *AnnouncementHandler) UpdateAnnouncement(c *gin.Context) {
	var req request.UpdateAnnouncementReq
	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.Fail(c, 400, "获取参数失败")
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
		response.Fail(c, 400, err.Error())
		return
	}
	response.Success(c, "公告更新成功")
	return
}

func (a *AnnouncementHandler) PushAnnouncement(c *gin.Context) {
	var req request.PushAnnouncementReq
	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.Fail(c, 400, "获取参数失败")
		return
	}
	userID := c.GetUint64("user_id")
	ctx := c.Request.Context()
	err = a.AnnouncementService.PushAnnouncement(ctx, req.AnnouncementID, userID)
	if err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.Success(c, "推送成功")
	return
}

func (a *AnnouncementHandler) GetAnnouncementList(c *gin.Context) {
	var req request.GetAnnouncementListReq
	err := c.ShouldBindQuery(&req)
	if err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	id := c.GetUint64("user_id")
	announcementInfos, err := a.AnnouncementService.GetAnnouncementInfoList(id, req.Status)
	if err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.Success(c, announcementInfos)
	return
}
func (a *AnnouncementHandler) DeleteAnnouncement(c *gin.Context) {
	userID := c.GetUint64("user_id")
	var req request.DeleteAnnouncementReq
	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.Fail(c, 400, "获取参数失败")
		return
	}
	err = a.AnnouncementService.DeleteAnnouncement(req.AnnouncementID, userID)
	if err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.Success(c, "删除公告成功")
	return
}
