package handler

import (
	"suseoaa/internal/request"
	"suseoaa/internal/service"
	"suseoaa/pkg/response"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	UserService service.UserService
}

func NewUserHandler(userService service.UserService) UserHandler {
	return UserHandler{UserService: userService}
}

func (u *UserHandler) GetInfo(c *gin.Context) {
	id := c.GetUint64("user_id")
	ctx := c.Request.Context()
	result, err := u.UserService.GetUserInfo(ctx, id)
	if err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.Success(c, result)
}

func (u *UserHandler) GetUserList(c *gin.Context) {
	var req request.UserListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Fail(c, 400, "获取query失败")
		return
	}
	userList, total, err := u.UserService.GetUserList(req.Keyword, req.Department, req.Role, req.Page, req.PageSize)
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}
	res := map[string]any{
		"total": total,
		"list":  userList,
	}
	response.Success(c, res)
}

func (u *UserHandler) UpdateUserInfo(c *gin.Context) {
	var req request.UpdateUserInfoReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "获取json失败")
		return
	}
	id := c.GetUint64("user_id")
	ctx := c.Request.Context()
	err := u.UserService.UpdateUserInfo(ctx, id, req.Username, req.Email, req.Avatar)
	if err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.Success(c, nil)
}
func (u *UserHandler) BatchUserInfo(c *gin.Context) {
	var req []request.BatchUserInfoReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "获取json失败")
		return
	}
	id := c.GetUint64("user_id")
	departmentID, roleID, err := u.UserService.GetDepartmentIDAndRoleIDByID(id)
	if err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	res, err := u.UserService.BatchUserInfo(req, departmentID, roleID)
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}
	response.Success(c, res)

}
func (u *UserHandler) DeleteUser(c *gin.Context) {
	id := c.GetUint64("user_id")
	var req request.DeleteUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	err := u.UserService.DeleteUser(id, req.UserID)
	if err != nil {
		response.Fail(c, 400, err.Error())
		return

	}
	response.Success(c, nil)
	return
}
