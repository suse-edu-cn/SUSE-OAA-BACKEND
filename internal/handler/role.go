package handler

import (
	"suseoaa/internal/service"
	"suseoaa/pkg/response"

	"github.com/gin-gonic/gin"
)

type RoleHandler struct {
	RoleService service.RoleService
}

func NewRoleHandler(roleService service.RoleService) RoleHandler {
	return RoleHandler{RoleService: roleService}
}

func (r RoleHandler) FindAll(c *gin.Context) {
	roles, err := r.RoleService.GetAll()
	if err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.Success(c, roles)
	return
}
