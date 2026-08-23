package handler

import (
	"suseoaa/internal/model"
	"suseoaa/internal/request"
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
func (r RoleHandler) Create(c *gin.Context) {
	var req request.CreateRoleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "获取参数失败")
		return
	}
	id := c.GetUint64("user_id")
	err := r.RoleService.Create(id, &model.Role{
		Name:  req.Name,
		Level: req.Level,
	})
	if err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.Success(c, nil)
	return
}
func (r RoleHandler) Update(c *gin.Context) {
	var req request.UpdateRoleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "获取参数失败")
		return
	}
	id := c.GetUint64("user_id")
	err := r.RoleService.Update(id, &model.Role{
		ID:    req.RoleID,
		Name:  req.Name,
		Level: req.Level,
	})
	if err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.Success(c, nil)
	return
}
