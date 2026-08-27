package handler

import (
	"suseoaa/internal/model"
	"suseoaa/internal/request"
	"suseoaa/internal/service"
	"suseoaa/pkg/response"

	"github.com/gin-gonic/gin"
)

type TermHandler struct {
	TermService service.TermService
}

func NewTermHandler(termService service.TermService) TermHandler {
	return TermHandler{
		TermService: termService,
	}
}

func (t *TermHandler) CreateTerm(c *gin.Context) {
	var req request.CreateTermReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "获取参数失败")
		return
	}
	id := c.GetUint64("user_id")
	err := t.TermService.CheckLevel(id)
	if err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	err = t.TermService.CreateTerm(model.Term{
		Title:        req.Title,
		Type:         req.Type,
		Year:         req.Year,
		EditStartAt:  req.EditPeriod.StartAt,
		EditEndAt:    req.EditPeriod.EndAt,
		QueryStartAt: req.QueryPeriod.StartAt,
		QueryEndAt:   req.QueryPeriod.EndAt,
	})
	if err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.Success(c, "term创建成功")
	return
}
func (t *TermHandler) UpdateTerm(c *gin.Context) {
	var req request.UpdateTermReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	id := c.GetUint64("user_id")
	err := t.TermService.CheckLevel(id)
	if err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	err = t.TermService.UpdateTerm(model.Term{
		ID:           req.ID,
		Title:        req.Title,
		EditStartAt:  req.EditPeriod.StartAt,
		EditEndAt:    req.EditPeriod.EndAt,
		QueryStartAt: req.QueryPeriod.StartAt,
		QueryEndAt:   req.QueryPeriod.EndAt,
	})
	if err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.Success(c, "term 更新成功")
	return
}
