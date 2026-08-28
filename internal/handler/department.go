package handler

import (
	"suseoaa/internal/model"
	"suseoaa/internal/request"
	"suseoaa/internal/service"
	"suseoaa/pkg/response"

	"github.com/gin-gonic/gin"
)

type DepartmentHandler struct {
	DepartmentService service.DepartmentService
}

func NewDepartmentHandler(departmentService service.DepartmentService) DepartmentHandler {
	return DepartmentHandler{
		DepartmentService: departmentService,
	}
}

func (d *DepartmentHandler) GetAll(c *gin.Context) {
	departments, err := d.DepartmentService.GetAll()
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}
	response.Success(c, departments)
}
func (d *DepartmentHandler) Create(c *gin.Context) {
	var req request.CreateDepartmentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "获取参数失败")
		return
	}
	id := c.GetUint64("user_id")
	err := req.CheckType()
	if err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	err = d.DepartmentService.CreateDepartment(id, &model.Department{
		Name: req.Name,
		Type: req.Type,
	})
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}
	response.Success(c, nil)
	return
}
func (d *DepartmentHandler) Update(c *gin.Context) {
	var req request.UpdateDepartmentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "获取参数失败")
		return
	}
	id := c.GetUint64("user_id")
	err := req.CheckType()
	if err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	err = d.DepartmentService.UpdateDepartment(id, &model.Department{
		ID:   req.DepartmentID,
		Name: req.Name,
		Type: req.Type,
	})
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}
	response.Success(c, nil)
	return
}
