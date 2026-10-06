package handler

import (
	"suseoaa/internal/model"
	"suseoaa/internal/request"
	"suseoaa/internal/service"
	"suseoaa/pkg/response"

	"github.com/gin-gonic/gin"
)

type AnnouncementHandler struct {
	AnnouncementService *service.AnnouncementService
}

func NewAnnouncementHandler(announcementService *service.AnnouncementService) *AnnouncementHandler {
	return &AnnouncementHandler{
		AnnouncementService: announcementService,
	}
}

func (a *AnnouncementHandler) CreateAnnouncement(c *gin.Context) {
	var req request.CreateAnnouncementReq
	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.BadRequest(c, "获取参数失败")
		return
	}
	userID := c.GetUint64("user_id")
	announcement := model.Announcement{
		Title:        req.Title,
		CreatedId:    userID,
		Content:      req.Content,
		DepartmentID: req.DepartmentID,
	}

	ctx := c.Request.Context()
	res, err := a.AnnouncementService.CreateAnnouncement(ctx, userID, announcement)
	if err != nil {
		if err.Error() == "权限不够" {
			response.Forbidden(c, "权限不足")
			return
		}
		response.ServerError(c, "创建公告", err)
		return
	}
	response.Success(c, res)
}

func (a *AnnouncementHandler) UpdateAnnouncement(c *gin.Context) {
	var req request.UpdateAnnouncementReq
	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.BadRequest(c, "获取参数失败")
		return
	}
	userID := c.GetUint64("user_id")
	announcement := model.Announcement{
		ID:      req.AnnouncementID,
		Title:   req.Title,
		Content: req.Content,
	}
	ctx := c.Request.Context()
	err = a.AnnouncementService.UpdateAnnouncement(ctx, userID, announcement)
	if err != nil {
		if err.Error() == "权限不够" {
			response.Forbidden(c, "权限不足")
			return
		}
		response.ServerError(c, "更新公告", err)
		return
	}
	response.Success(c, "公告更新成功")
}

func (a *AnnouncementHandler) PushAnnouncement(c *gin.Context) {
	var req request.PushAnnouncementReq
	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.BadRequest(c, "获取参数失败")
		return
	}
	userID := c.GetUint64("user_id")
	ctx := c.Request.Context()
	err = a.AnnouncementService.PushAnnouncement(ctx, req.AnnouncementID, userID)
	if err != nil {
		if err.Error() == "权限不够" {
			response.Forbidden(c, "权限不足")
			return
		}
		response.ServerError(c, "推送公告", err)
		return
	}
	response.Success(c, "推送成功")
}

func (a *AnnouncementHandler) GetAnnouncementList(c *gin.Context) {
	var req request.GetAnnouncementListReq
	err := c.ShouldBindQuery(&req)
	if err != nil {
		response.BadRequest(c, err.Error())
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
		response.ServerError(c, "获取公告列表", err)
		return
	}
	response.Success(c, announcementInfos)
}

func (a *AnnouncementHandler) GetAnnouncement(c *gin.Context) {
	var req request.GetAnnouncementReq
	err := c.ShouldBindQuery(&req)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	userID := c.GetUint64("user_id")
	ctx := c.Request.Context()
	announcement, err := a.AnnouncementService.GetAnnouncementInfo(ctx, userID, req.AnnouncementID)
	if err != nil {
		response.ServerError(c, "获取公告详情", err)
		return
	}
	response.Success(c, announcement)
}

func (a *AnnouncementHandler) DeleteAnnouncement(c *gin.Context) {
	userID := c.GetUint64("user_id")
	var req request.DeleteAnnouncementReq
	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.BadRequest(c, "获取参数失败")
		return
	}
	ctx := c.Request.Context()
	err = a.AnnouncementService.DeleteAnnouncement(ctx, req.AnnouncementID, userID)
	if err != nil {
		if err.Error() == "权限不够" {
			response.Forbidden(c, "权限不足")
			return
		}
		response.ServerError(c, "删除公告", err)
		return
	}
	response.Success(c, "删除公告成功")
}
