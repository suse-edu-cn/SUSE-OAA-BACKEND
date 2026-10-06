package handler

import (
	"suseoaa/internal/request"
	"suseoaa/internal/service"
	"suseoaa/pkg/response"
	"suseoaa/pkg/utils"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	UserService    *service.UserService
	JwtSecret      string
	JwtExpire      int
	JwtRefreshTime uint
}

func NewAuthHandler(userService *service.UserService, JwtSecret string, jwtExpire int, refreshTime uint) *AuthHandler {
	return &AuthHandler{
		UserService:    userService,
		JwtSecret:      JwtSecret,
		JwtExpire:      jwtExpire,
		JwtRefreshTime: refreshTime,
	}
}

func (a *AuthHandler) Login(c *gin.Context) {
	var req request.LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "获取json失败")
		return
	}
	ctx := c.Request.Context()
	user, refreshToken, err := a.UserService.Login(ctx, req, a.JwtRefreshTime)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	token, err := utils.GenerateToken(user.Name, user.ID, user.StudentID, a.JwtSecret, a.JwtExpire)
	if err != nil {
		response.ServerError(c, "生成Token", err)
		return
	}
	res := map[string]string{
		"token":         token,
		"refresh_token": refreshToken,
	}
	response.Success(c, res)
}
func (a *AuthHandler) Refresh(c *gin.Context) {
	var req request.RefreshReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "获取json失败")
		return
	}
	ctx := c.Request.Context()
	refreshToken, err := a.UserService.GetRefreshToken(ctx, req.UserID, req.Device)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if refreshToken != req.RefreshToken {
		response.BadRequest(c, "refresh token 错误")
		return
	}
	user, err := a.UserService.FindUserByID(ctx, req.UserID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	token, err := utils.GenerateToken(user.Name, user.ID, user.StudentID, a.JwtSecret, a.JwtExpire)
	if err != nil {
		response.ServerError(c, "生成Token", err)
		return
	}
	res := map[string]string{
		"token":         token,
		"refresh_token": refreshToken,
	}
	response.Success(c, res)
}

func (a *AuthHandler) Register(c *gin.Context) {
	var req request.RegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "获取json失败")
		return
	}
	ctx := c.Request.Context()
	if err := a.UserService.Register(ctx, req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, nil)
}

func (a *AuthHandler) Logout(c *gin.Context) {
	var req request.LogoutReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "获取json失败")
		return
	}
	id := c.GetUint64("user_id")
	ctx := c.Request.Context()

	err := a.UserService.DeleteRefreshToken(ctx, id, req.Device)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, nil)
}

func (a *AuthHandler) UpdatePassword(c *gin.Context) {
	var req request.UpdatePasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "获取json失败")
		return
	}
	id := c.GetUint64("user_id")
	ctx := c.Request.Context()
	err := a.UserService.UpdatePassword(ctx, id, req.OldPassword, req.NewPassword)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, nil)
}
func (a *AuthHandler) SendVerificationCode(c *gin.Context) {
	var req request.SendVerificationCodeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "获取json失败")
		return
	}

	ctx := c.Request.Context()
	err := a.UserService.SendVerificationCode(ctx, req.Account, req.Scene)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, nil)
}

func (a *AuthHandler) ResetPassword(c *gin.Context) {
	var req request.ResetPasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "获取json失败")
		return
	}

	ctx := c.Request.Context()
	err := a.UserService.ResetPassword(ctx, req.Account, req.Code, req.Password)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, nil)
}
