package handler

import (
	"suseoaa/internal/model"
	"suseoaa/internal/request"
	"suseoaa/internal/service"
	"suseoaa/pkg/response"

	"github.com/gin-gonic/gin"
)

type TermHandler struct {
	TermService *service.TermService
}

func NewTermHandler(termService *service.TermService) *TermHandler {
	return &TermHandler{
		TermService: termService,
	}
}

// 业务周期

func (t *TermHandler) CreateTerm(c *gin.Context) {
	var req request.CreateTermReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "获取参数失败", nil)
		return
	}
	id := c.GetUint64("user_id")
	ctx := c.Request.Context()
	err := t.TermService.CheckLevel(ctx, id)
	if err != nil {
		response.Fail(c, 400, err.Error(), nil)
		return
	}
	err = t.TermService.CreateTerm(ctx, model.Term{
		Title:        req.Title,
		Type:         req.Type,
		Year:         req.Year,
		EditStartAt:  req.EditPeriod.StartAt,
		EditEndAt:    req.EditPeriod.EndAt,
		QueryStartAt: req.QueryPeriod.StartAt,
		QueryEndAt:   req.QueryPeriod.EndAt,
	})
	if err != nil {
		response.Fail(c, 400, err.Error(), nil)
		return
	}
	response.Success(c, "term创建成功")
}

func (t *TermHandler) UpdateTerm(c *gin.Context) {
	var req request.UpdateTermReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, err.Error(), nil)
		return
	}
	id := c.GetUint64("user_id")
	ctx := c.Request.Context()
	err := t.TermService.CheckLevel(ctx, id)
	if err != nil {
		response.Fail(c, 400, err.Error(), nil)
		return
	}
	err = t.TermService.UpdateTerm(ctx, model.Term{
		ID:           req.TermID,
		Title:        req.Title,
		EditStartAt:  req.EditPeriod.StartAt,
		EditEndAt:    req.EditPeriod.EndAt,
		QueryStartAt: req.QueryPeriod.StartAt,
		QueryEndAt:   req.QueryPeriod.EndAt,
	})
	if err != nil {
		response.Fail(c, 400, err.Error(), nil)
		return
	}
	response.Success(c, "term 更新成功")
}

func (t *TermHandler) GetTermList(c *gin.Context) {
	var req request.GetTermListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Fail(c, 400, "获取参数失败: "+err.Error(), nil)
		return
	}
	ctx := c.Request.Context()
	termList, err := t.TermService.GetTermList(ctx, req.Year, req.Type)
	if err != nil {
		response.Fail(c, 400, err.Error(), nil)
		return
	}
	response.Success(c, termList)
}

func (t *TermHandler) DeleteTerm(c *gin.Context) {
	var req request.DeleteTermReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, err.Error(), nil)
		return
	}

	id := c.GetUint64("user_id")
	ctx := c.Request.Context()
	if err := t.TermService.DeleteTerm(ctx, id, req.TermID); err != nil {
		response.Fail(c, 400, err.Error(), nil)
		return
	}

	response.Success(c, "term 删除成功")
}

// 申请表

func (t *TermHandler) CreateApplication(c *gin.Context) {
	var req request.CreateApplicationReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "获取参数失败: "+err.Error(), nil)
		return
	}
	id := c.GetUint64("user_id")
	studentID := c.GetString("student_id")
	name := c.GetString("name")
	ctx := c.Request.Context()
	err := t.TermService.CreateApplication(ctx, model.Application{
		TermID:          req.TermID,
		UserID:          id,
		Name:            name,
		Gender:          req.Gender,
		AvatarURI:       req.Avatar,
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
		response.Fail(c, 400, err.Error(), nil)
		return
	}
	response.Success(c, "创建成功")
}

func (t *TermHandler) UpdateApplication(c *gin.Context) {
	var req request.UpdateApplicationReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, err.Error(), nil)
		return
	}
	id := c.GetUint64("user_id")
	ctx := c.Request.Context()
	err := t.TermService.UpdateApplication(ctx, model.Application{
		UserID:          id,
		Gender:          req.Gender,
		AvatarURI:       req.Avatar,
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
		response.Fail(c, 400, err.Error(), nil)
		return
	}
	response.Success(c, "更新成功")
}

func (t *TermHandler) GetMyApplications(c *gin.Context) {
	id := c.GetUint64("user_id")
	ctx := c.Request.Context()
	application, err := t.TermService.GetMyApplications(ctx, id)
	if err != nil {
		response.Fail(c, 400, err.Error(), nil)
		return
	}
	response.Success(c, application)
}

func (t *TermHandler) GetApplicationDepartmentList(c *gin.Context) {
	var req request.GetApplicationDepartmentReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Fail(c, 400, err.Error(), nil)
		return
	}
	ctx := c.Request.Context()
	departments, err := t.TermService.GetDepartmentByRoleID(ctx, req.RoleID)
	if err != nil {
		response.Fail(c, 400, err.Error(), nil)
		return
	}
	response.Success(c, departments)
}

func (t *TermHandler) GetApplicationRoleList(c *gin.Context) {
	var req request.GetApplicationRoleReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Fail(c, 400, err.Error(), nil)
		return
	}
	ctx := c.Request.Context()
	roles, err := t.TermService.GetRolesByDepartmentsID(ctx, req.DepartmentID)
	if err != nil {
		response.Fail(c, 400, err.Error(), nil)
		return
	}
	response.Success(c, roles)
}

func (t *TermHandler) GetApplicationList(c *gin.Context) {
	var req request.GetApplicationListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Fail(c, 400, err.Error(), nil)
		return
	}
	id := c.GetUint64("user_id")
	ctx := c.Request.Context()
	applications, err := t.TermService.GetApplicationList(ctx, id, req.DepartmentID, req.TermID)
	if err != nil {
		response.Fail(c, 400, err.Error(), nil)
		return
	}
	response.Success(c, applications)
}

func (t *TermHandler) DeleteApplication(c *gin.Context) {
	id := c.GetUint64("user_id")
	var req request.DeleteApplicationReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, err.Error(), nil)
		return
	}
	ctx := c.Request.Context()
	err := t.TermService.DeleteApplication(ctx, req.ApplicationID, id)
	if err != nil {
		response.Fail(c, 400, err.Error(), nil)
		return
	}
	response.Success(c, "删除申请表成功")
}

// 面试官

func (t *TermHandler) CreateInterviewers(c *gin.Context) {
	var req request.CreateInterviewer
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, err.Error(), nil)
		return
	}
	id := c.GetUint64("user_id")
	ctx := c.Request.Context()
	err := t.TermService.CreateInterviewers(ctx, id, req)
	if err != nil {
		response.Fail(c, 400, err.Error(), nil)
		return
	}
	response.Success(c, "ok")
}

func (t *TermHandler) GetInterviewerList(c *gin.Context) {
	var req request.GetInterviewerListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Fail(c, 400, err.Error(), nil)
		return
	}
	id := c.GetUint64("user_id")
	ctx := c.Request.Context()
	interviewerList, err := t.TermService.GetInterviewerList(ctx, id, req.TermID)
	if err != nil {
		response.Fail(c, 400, err.Error(), nil)
		return
	}
	response.Success(c, interviewerList)
}

func (t *TermHandler) UpdateInterviewer(c *gin.Context) {
	var req request.UpdateInterviewer
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, err.Error(), nil)
		return
	}
	id := c.GetUint64("user_id")
	ctx := c.Request.Context()
	err := t.TermService.UpdateInterviewer(ctx, id, req)
	if err != nil {
		response.Fail(c, 400, err.Error(), nil)
		return
	}
	response.Success(c, "更新成功")
}

func (t *TermHandler) DeleteInterviewer(c *gin.Context) {
	var req request.DeleteInterviewer
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, err.Error(), nil)
		return
	}

	id := c.GetUint64("user_id")
	ctx := c.Request.Context()
	if err := t.TermService.DeleteInterviewer(ctx, id, req.InterviewerID); err != nil {
		response.Fail(c, 400, err.Error(), nil)
		return
	}

	response.Success(c, "删除面试官成功")
}

// 面试结果

func (t *TermHandler) CreateInterviewResult(c *gin.Context) {
	var req request.CreateInterviewResultReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, err.Error(), nil)
		return
	}
	id := c.GetUint64("user_id")
	ctx := c.Request.Context()
	err := t.TermService.CreateInterviewResult(ctx, id, req)
	if err != nil {
		response.Fail(c, 400, err.Error(), nil)
		return
	}
	response.Success(c, "创建面试结果成功")
}

func (t *TermHandler) UpdateInterviewResult(c *gin.Context) {
	var req request.UpdateInterviewResultReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, err.Error(), nil)
		return
	}

	operatorID := c.GetUint64("user_id")
	ctx := c.Request.Context()
	if err := t.TermService.UpdateInterviewResult(ctx, operatorID, req); err != nil {
		response.Fail(c, 400, err.Error(), nil)
		return
	}

	response.Success(c, "更新面试结果成功")
}

func (t *TermHandler) GetInterviewResultList(c *gin.Context) {
	var req request.GetInterviewResultListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Fail(c, 400, err.Error(), nil)
		return
	}

	operatorID := c.GetUint64("user_id")
	ctx := c.Request.Context()
	results, err := t.TermService.GetInterviewResultList(ctx, operatorID, req.TermID)
	if err != nil {
		response.Fail(c, 400, err.Error(), nil)
		return
	}

	response.Success(c, results)
}

func (t *TermHandler) GetInterviewResultDecision(c *gin.Context) {
	result := t.TermService.GetInterviewResultDecision()
	response.Success(c, result)
}
