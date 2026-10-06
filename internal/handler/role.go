package handler

import (
	"strings"
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
		response.ServerError(c, "获取角色列表", err)
		return
	}
	response.Success(c, roles)
}

func (r *RoleHandler) Create(c *gin.Context) {
	var req request.CreateRoleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "获取参数失败")
		return
	}
	id := c.GetUint64("user_id")
	err := req.CheckType()
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	ctx := c.Request.Context()
	err = r.RoleService.Create(ctx, id, &model.Role{
		Name:  req.Name,
		Level: req.Level,
		Type:  req.Type,
	})
	if err != nil {
		if err.Error() == "权限不够" {
			response.Forbidden(c, "权限不足")
			return
		}
		response.ServerError(c, "创建角色", err)
		return
	}
	response.Success(c, nil)
}

func (r *RoleHandler) Update(c *gin.Context) {
	var req request.UpdateRoleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "获取参数失败")
		return
	}
	id := c.GetUint64("user_id")
	err := req.CheckType()
	if err != nil {
		response.BadRequest(c, err.Error())
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
		if err.Error() == "权限不够" || strings.Contains(err.Error(), "不能修改") || strings.Contains(err.Error(), "不能把目标") {
			response.Forbidden(c, err.Error())
			return
		}
		response.ServerError(c, "更新角色", err)
		return
	}
	response.Success(c, nil)
}
