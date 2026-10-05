package handler

import (
	"suseoaa/internal/model"
	"suseoaa/internal/request"
	"suseoaa/internal/service"
	"suseoaa/pkg/response"

	"github.com/gin-gonic/gin"
)

type RoleHandler struct {
	RoleService *service.RoleService
}

func NewRoleHandler(roleService *service.RoleService) *RoleHandler {
	return &RoleHandler{RoleService: roleService}
}

func (r *RoleHandler) FindAll(c *gin.Context) {
	ctx := c.Request.Context()
	roles, err := r.RoleService.GetAll(ctx)
	if err != nil {
		response.Fail(c, 400, err.Error(), nil)
		return
	}
	response.Success(c, roles)
}

func (r *RoleHandler) Create(c *gin.Context) {
	var req request.CreateRoleReq
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
	err = r.RoleService.Create(ctx, id, &model.Role{
		Name:  req.Name,
		Level: req.Level,
		Type:  req.Type,
	})
	if err != nil {
		response.Fail(c, 400, err.Error(), nil)
		return
	}
	response.Success(c, nil)
}

func (r *RoleHandler) Update(c *gin.Context) {
	var req request.UpdateRoleReq
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
	err = r.RoleService.Update(ctx, id, &model.Role{
		ID:    req.RoleID,
		Name:  req.Name,
		Level: req.Level,
		Type:  req.Type,
	}, req.IsActive)
	if err != nil {
		response.Fail(c, 400, err.Error(), nil)
		return
	}
	response.Success(c, nil)
}
