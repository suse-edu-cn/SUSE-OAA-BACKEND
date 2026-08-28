package handler

import (
	"fmt"
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

//----------------------------
//业务周期

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
func (t *TermHandler) GetTermList(c *gin.Context) {
	var req request.GetTermListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Fail(c, 400, "获取参数失败: "+err.Error())
		return
	}
	id := c.GetUint64("user_id")
	err := t.TermService.CheckLevel(id)
	if err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	fmt.Println(req)
	termList, err := t.TermService.GetTermList(req.Year, req.Type)
	if err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.Success(c, termList)
	return
}

//-------------------------------
//申请表

func (t *TermHandler) CreateApplication(c *gin.Context) {
	var req request.CreateApplicationReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "获取参数失败: "+err.Error())
		return
	}
	id := c.GetUint64("user_id")
	studentID := c.GetString("student_id")
	name := c.GetString("name")
	err := t.TermService.CreateApplication(model.Application{
		TermID:          req.TermID,
		UserID:          id,
		Name:            name,
		Gender:          req.Gender,
		StudentID:       studentID,
		College:         req.College,
		MajorClass:      req.MajorClass,
		PoliticalStatus: req.PoliticalStatus,
		BirthDate:       req.BirthDate,
		QQ:              req.QQ,
		Phone:           req.Phone,
		FirstChoice: model.OrganizationRole{
			DepartmentID: req.FirstChoice.DepartmentID,
			RoleID:       req.FirstChoice.RoleID,
		},
		SecondChoice: model.OrganizationRole{
			DepartmentID: req.SecondChoice.DepartmentID,
			RoleID:       req.SecondChoice.RoleID,
		},
		AllowAdjust: req.AllowAdjust,
		Resume:      req.Resume,
		Reason:      req.Reason,
	})
	if err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.Success(c, "创建成功")
	return
}
