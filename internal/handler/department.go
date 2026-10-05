package handler

import (
	"suseoaa/internal/model"
	"suseoaa/internal/request"
	"suseoaa/internal/service"
	"suseoaa/pkg/response"

	"github.com/gin-gonic/gin"
)

type DepartmentHandler struct {
	DepartmentService *service.DepartmentService
}

func NewDepartmentHandler(departmentService *service.DepartmentService) *DepartmentHandler {
	return &DepartmentHandler{
		DepartmentService: departmentService,
	}
}

func (d *DepartmentHandler) GetAll(c *gin.Context) {
	ctx := c.Request.Context()
	departments, err := d.DepartmentService.GetAll(ctx)
	if err != nil {
		response.Fail(c, 500, err.Error(), nil)
		return
	}
	response.Success(c, departments)
}

func (d *DepartmentHandler) Create(c *gin.Context) {
	var req request.CreateDepartmentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "获取参数失败", nil)
		return
	}
	id := c.GetUint64("user_id")
	err := req.CheckType()
	if err != nil {
		response.Fail(c, 400, err.Error(), nil)
		return
	}
	ctx := c.Request.Context()
	err = d.DepartmentService.CreateDepartment(ctx, id, &model.Department{
		Name: req.Name,
		Type: req.Type,
	})
	if err != nil {
		response.Fail(c, 500, err.Error(), nil)
		return
	}
	response.Success(c, nil)
}

func (d *DepartmentHandler) Update(c *gin.Context) {
	var req request.UpdateDepartmentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, err.Error(), nil)
		return
	}
	id := c.GetUint64("user_id")
	err := req.CheckType()
	if err != nil {
		response.Fail(c, 400, err.Error(), nil)
		return
	}
	ctx := c.Request.Context()
	err = d.DepartmentService.UpdateDepartment(ctx, id, &model.Department{
		ID:   req.DepartmentID,
		Name: req.Name,
		Type: req.Type,
	}, req.IsActive)
	if err != nil {
		response.Fail(c, 500, err.Error(), nil)
		return
	}
	response.Success(c, nil)
}
