package handler

import (
	"strings"
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

func handleTermError(c *gin.Context, action string, err error) {
	if err == nil {
		return
	}
	msg := err.Error()
	if strings.Contains(msg, "权限") || strings.Contains(msg, "无权") {
		response.Forbidden(c, "权限不足")
		return
	}
	if strings.Contains(msg, "周期") ||
		strings.Contains(msg, "时间") ||
		strings.Contains(msg, "申请") ||
		strings.Contains(msg, "面试") ||
		strings.Contains(msg, "职位") ||
		strings.Contains(msg, "志愿") ||
		strings.Contains(msg, "调剂") ||
		strings.Contains(msg, "决定") ||
		strings.Contains(msg, "存在") ||
		strings.Contains(msg, "不匹配") ||
		strings.Contains(msg, "类型错误") {
		response.BadRequest(c, msg)
		return
	}
	response.ServerError(c, action, err)
}

// 业务周期

func (t *TermHandler) CreateTerm(c *gin.Context) {
	var req request.CreateTermReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "获取参数失败")
		return
	}
	id := c.GetUint64("user_id")
	ctx := c.Request.Context()
	err := t.TermService.CheckLevel(ctx, id)
	if err != nil {
		handleTermError(c, "检查权限", err)
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
		handleTermError(c, "创建招新周期", err)
		return
	}
	response.Success(c, "term创建成功")
}

func (t *TermHandler) UpdateTerm(c *gin.Context) {
	var req request.UpdateTermReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "获取参数失败")
		return
	}
	id := c.GetUint64("user_id")
	ctx := c.Request.Context()
	err := t.TermService.CheckLevel(ctx, id)
	if err != nil {
		handleTermError(c, "检查权限", err)
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
		handleTermError(c, "更新招新周期", err)
		return
	}
	response.Success(c, "term 更新成功")
}

func (t *TermHandler) GetTermList(c *gin.Context) {
	var req request.GetTermListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "获取参数失败: "+err.Error())
		return
	}
	ctx := c.Request.Context()
	termList, err := t.TermService.GetTermList(ctx, req.Year, req.Type)
	if err != nil {
		handleTermError(c, "获取招新周期列表", err)
		return
	}
	response.Success(c, termList)
}

func (t *TermHandler) DeleteTerm(c *gin.Context) {
	var req request.DeleteTermReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "获取参数失败")
		return
	}

	id := c.GetUint64("user_id")
	ctx := c.Request.Context()
	if err := t.TermService.DeleteTerm(ctx, id, req.TermID); err != nil {
		handleTermError(c, "删除招新周期", err)
		return
	}

	response.Success(c, "term 删除成功")
}

// 申请表

func (t *TermHandler) CreateApplication(c *gin.Context) {
	var req request.CreateApplicationReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "获取参数失败: "+err.Error())
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
		handleTermError(c, "创建申请表", err)
		return
	}
	response.Success(c, "创建成功")
}

func (t *TermHandler) UpdateApplication(c *gin.Context) {
	var req request.UpdateApplicationReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "获取参数失败")
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
		handleTermError(c, "更新申请表", err)
		return
	}
	response.Success(c, "更新成功")
}

func (t *TermHandler) GetMyApplications(c *gin.Context) {
	id := c.GetUint64("user_id")
	ctx := c.Request.Context()
	application, err := t.TermService.GetMyApplications(ctx, id)
	if err != nil {
		handleTermError(c, "获取我的申请表", err)
		return
	}
	response.Success(c, application)
}

func (t *TermHandler) GetApplicationDepartmentList(c *gin.Context) {
	var req request.GetApplicationDepartmentReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	ctx := c.Request.Context()
	departments, err := t.TermService.GetDepartmentByRoleID(ctx, req.RoleID)
	if err != nil {
		handleTermError(c, "获取申请部门列表", err)
		return
	}
	response.Success(c, departments)
}

func (t *TermHandler) GetApplicationRoleList(c *gin.Context) {
	var req request.GetApplicationRoleReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	ctx := c.Request.Context()
	roles, err := t.TermService.GetRolesByDepartmentsID(ctx, req.DepartmentID)
	if err != nil {
		handleTermError(c, "获取申请职位列表", err)
		return
	}
	response.Success(c, roles)
}

func (t *TermHandler) GetApplicationList(c *gin.Context) {
	var req request.GetApplicationListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	id := c.GetUint64("user_id")
	ctx := c.Request.Context()
	applications, err := t.TermService.GetApplicationList(ctx, id, req.DepartmentID, req.TermID)
	if err != nil {
		handleTermError(c, "获取申请表列表", err)
		return
	}
	response.Success(c, applications)
}

func (t *TermHandler) DeleteApplication(c *gin.Context) {
	id := c.GetUint64("user_id")
	var req request.DeleteApplicationReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	ctx := c.Request.Context()
	err := t.TermService.DeleteApplication(ctx, req.ApplicationID, id)
	if err != nil {
		handleTermError(c, "删除申请表", err)
		return
	}
	response.Success(c, "删除申请表成功")
}

// 面试官

func (t *TermHandler) CreateInterviewers(c *gin.Context) {
	var req request.CreateInterviewer
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	id := c.GetUint64("user_id")
	ctx := c.Request.Context()
	err := t.TermService.CreateInterviewers(ctx, id, req)
	if err != nil {
		handleTermError(c, "添加面试官", err)
		return
	}
	response.Success(c, "ok")
}

func (t *TermHandler) GetInterviewerList(c *gin.Context) {
	var req request.GetInterviewerListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	id := c.GetUint64("user_id")
	ctx := c.Request.Context()
	interviewerList, err := t.TermService.GetInterviewerList(ctx, id, req.TermID)
	if err != nil {
		handleTermError(c, "获取面试官列表", err)
		return
	}
	response.Success(c, interviewerList)
}

func (t *TermHandler) UpdateInterviewer(c *gin.Context) {
	var req request.UpdateInterviewer
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	id := c.GetUint64("user_id")
	ctx := c.Request.Context()
	err := t.TermService.UpdateInterviewer(ctx, id, req)
	if err != nil {
		handleTermError(c, "更新面试官", err)
		return
	}
	response.Success(c, "更新成功")
}

func (t *TermHandler) DeleteInterviewer(c *gin.Context) {
	var req request.DeleteInterviewer
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	id := c.GetUint64("user_id")
	ctx := c.Request.Context()
	if err := t.TermService.DeleteInterviewer(ctx, id, req.InterviewerID); err != nil {
		handleTermError(c, "删除面试官", err)
		return
	}

	response.Success(c, "删除面试官成功")
}

// 面试结果

func (t *TermHandler) CreateInterviewResult(c *gin.Context) {
	var req request.CreateInterviewResultReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	id := c.GetUint64("user_id")
	ctx := c.Request.Context()
	err := t.TermService.CreateInterviewResult(ctx, id, req)
	if err != nil {
		handleTermError(c, "创建面试结果", err)
		return
	}
	response.Success(c, "创建面试结果成功")
}

func (t *TermHandler) UpdateInterviewResult(c *gin.Context) {
	var req request.UpdateInterviewResultReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	operatorID := c.GetUint64("user_id")
	ctx := c.Request.Context()
	if err := t.TermService.UpdateInterviewResult(ctx, operatorID, req); err != nil {
		handleTermError(c, "更新面试结果", err)
		return
	}

	response.Success(c, "更新面试结果成功")
}

func (t *TermHandler) GetInterviewResultList(c *gin.Context) {
	var req request.GetInterviewResultListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	operatorID := c.GetUint64("user_id")
	ctx := c.Request.Context()
	results, err := t.TermService.GetInterviewResultList(ctx, operatorID, req.TermID)
	if err != nil {
		handleTermError(c, "获取面试结果列表", err)
		return
	}

	response.Success(c, results)
}

func (t *TermHandler) GetInterviewResultDecision(c *gin.Context) {
	result := t.TermService.GetInterviewResultDecision()
	response.Success(c, result)
}
