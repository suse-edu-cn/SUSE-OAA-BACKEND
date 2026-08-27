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
	err := t.TermService.CreateTerm(id, model.Term{
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
	response.Success(c, "创建成功")
	return
}
