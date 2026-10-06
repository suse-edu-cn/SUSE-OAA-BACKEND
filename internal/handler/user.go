package handler

import (
	"strconv"
	"strings"
	"time"

	"suseoaa/internal/request"
	"suseoaa/internal/service"
	"suseoaa/pkg/response"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	UserService *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{UserService: userService}
}

func (u *UserHandler) GetInfo(c *gin.Context) {
	id := c.GetUint64("user_id")
	ctx := c.Request.Context()
	result, err := u.UserService.GetUserInfo(ctx, id)
	if err != nil {
		response.ServerError(c, "获取用户信息", err)
		return
	}
	response.Success(c, result)
}

func parseUserListIDOrName(id uint64, value string) (uint64, string) {
	value = strings.TrimSpace(value)
	if id != 0 || value == "" {
		return id, value
	}

	parsedID, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, value
	}
	return parsedID, ""
}

func (u *UserHandler) GetUserList(c *gin.Context) {
	var req request.UserListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "获取query失败")
		return
	}
	departmentID, departmentName := parseUserListIDOrName(req.DepartmentID, req.Department)
	roleID, roleName := parseUserListIDOrName(req.RoleID, req.Role)

	ctx := c.Request.Context()
	userList, total, err := u.UserService.GetUserList(ctx, req.Keyword, departmentName, roleName, departmentID, roleID, req.Page, req.PageSize, req.IsAll)
	if err != nil {
		response.ServerError(c, "获取用户列表", err)
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
		response.BadRequest(c, "获取json失败")
		return
	}
	id := c.GetUint64("user_id")
	ctx := c.Request.Context()
	err := u.UserService.UpdateUserInfo(ctx, id, req.Username, req.Email, req.Avatar)
	if err != nil {
		if strings.Contains(err.Error(), "头像") || strings.Contains(err.Error(), "占用") {
			response.BadRequest(c, err.Error())
			return
		}
		response.ServerError(c, "更新用户信息", err)
		return
	}
	response.Success(c, nil)
}
func (u *UserHandler) BatchUserInfo(c *gin.Context) {
	var req []request.BatchUserInfoReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "获取json失败")
		return
	}
	id := c.GetUint64("user_id")
	ctx := c.Request.Context()
	departmentID, roleID, err := u.UserService.GetDepartmentIDAndRoleIDByID(ctx, id)
	if err != nil {
		response.ServerError(c, "获取用户身份", err)
		return
	}
	res, err := u.UserService.BatchUserInfo(ctx, req, departmentID, roleID)
	if err != nil {
		if res == nil {
			response.BadRequest(c, err.Error())
		} else {
			response.BadRequestWithData(c, "部分错误", res)
		}
		return
	}
	response.Success(c, res)

}
func (u *UserHandler) DeleteUser(c *gin.Context) {
	id := c.GetUint64("user_id")
	var req request.DeleteUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	ctx := c.Request.Context()
	scheduledDeleteAt, err := u.UserService.DeleteUser(ctx, id, req.UserID, req.Code)
	if err != nil {
		if err.Error() == "权限不够" {
			response.Forbidden(c, "权限不足")
			return
		}
		if strings.Contains(err.Error(), "参数错误") || strings.Contains(err.Error(), "验证码") || strings.Contains(err.Error(), "冷静期") || strings.Contains(err.Error(), "注销本人账号") || strings.Contains(err.Error(), "用户不存在") {
			response.BadRequest(c, err.Error())
			return
		}
		response.ServerError(c, "注销用户", err)
		return
	}
	if scheduledDeleteAt != nil {
		response.BadRequestWithData(c, "账号处于冷静期", scheduledDeleteAt.In(time.FixedZone("Asia/Shanghai", 8*60*60)).Format("2006-01-02 15:04:05"))
		return
	}
	response.Success(c, nil)
}

func (u *UserHandler) CancelDeleteUser(c *gin.Context) {
	id := c.GetUint64("user_id")
	var req request.CancelDeleteUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "获取json失败")
		return
	}
	ctx := c.Request.Context()
	if err := u.UserService.CancelDeleteUser(ctx, id, req.Code); err != nil {
		if strings.Contains(err.Error(), "冷静期") || strings.Contains(err.Error(), "验证码") || strings.Contains(err.Error(), "用户不存在") {
			response.BadRequest(c, err.Error())
			return
		}
		response.ServerError(c, "取消注销账号", err)
		return
	}
	response.Success(c, nil)
}

