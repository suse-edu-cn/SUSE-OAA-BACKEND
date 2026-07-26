package handler

import (
	"growthos/internal/service"
	"growthos/pkg/response"

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
