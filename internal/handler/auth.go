package handler

import (
	"suseoaa/internal/request"
	"suseoaa/internal/service"
	"suseoaa/pkg/response"
	"suseoaa/pkg/utils"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	UserService    service.UserService
	JwtSecret      string
	JwtExpire      int
	JwtRefreshTime uint
}

func NewAuthHandler(userService service.UserService, JwtSecret string, jwtExpire int, refreshTime uint) AuthHandler {
	return AuthHandler{
		UserService:    userService,
		JwtSecret:      JwtSecret,
		JwtExpire:      jwtExpire,
		JwtRefreshTime: refreshTime,
	}
}

func (a *AuthHandler) Login(c *gin.Context) {
	var req request.LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "获取json失败")
		return
	}
	user, refreshToken, err := a.UserService.Login(req, a.JwtRefreshTime)
	if err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	var departmentID uint64
	var roleID uint64
	if user.DepartmentID != nil {
		departmentID = *user.DepartmentID
	}
	if user.RoleID != 0 {
		roleID = user.RoleID
	}
	token, err := utils.GenerateToken(user.Username, user.ID, departmentID, roleID, a.JwtSecret, a.JwtExpire)
	if err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	res := map[string]string{
		"token":        token,
		"refreshToken": refreshToken,
	}
	response.Success(c, res)
}
func (a *AuthHandler) Refresh(c *gin.Context) {
	var req request.RefreshReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "获取json失败")
		return
	}
	refreshToken, err := a.UserService.GetRefreshToken(req.UserId, req.Device)
	if err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	if refreshToken != req.RefreshToken {
		response.Fail(c, 400, "refresh token 错误")
		return
	}
	user, err := a.UserService.FindUserByID(req.UserId)
	if err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	var departmentID uint64
	var roleID uint64
	if user.DepartmentID != nil {
		departmentID = *user.DepartmentID
	}
	if user.RoleID != 0 {
		roleID = user.RoleID
	}
	token, err := utils.GenerateToken(user.Username, user.ID, departmentID, roleID, a.JwtSecret, a.JwtExpire)
	if err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	res := map[string]string{
		"token":        token,
		"refreshToken": refreshToken,
	}
	response.Success(c, res)
}

func (a *AuthHandler) Register(c *gin.Context) {
	var req request.RegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "获取json失败")
		return
	}
	if err := a.UserService.Register(req); err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.Success(c, nil)
}

func (a *AuthHandler) Logout(c *gin.Context) {
	var req request.LogoutReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "获取json失败")
		return
	}
	id := c.GetUint64("user_id")

	err := a.UserService.DeleteRefreshToken(id, req.Device)
	if err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.Success(c, nil)
}

func (a *AuthHandler) UpdatePassword(c *gin.Context) {
	var req request.UpdatePasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "获取json失败")
		return
	}
	id := c.GetUint64("user_id")
	err := a.UserService.UpdatePassword(id, req.OldPassword, req.NewPassword1, req.NewPassword2)
	if err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.Success(c, nil)
}
func (a *AuthHandler) SendVerificationCode(c *gin.Context) {
	var req request.SendVerificationCodeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "获取json失败")
		return
	}
	id := c.GetUint64("user_id")
	err := a.UserService.SendVerificationCode(id, req.Type)
	if err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.Success(c, nil)
}
func (a *AuthHandler) ResetPassword(c *gin.Context) {
	var req request.ResetPasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "获取json失败")
		return
	}
	id := c.GetUint64("user_id")
	err := a.UserService.ResetPassword(id, req.Code, req.Type)
	if err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.Success(c, nil)

}
