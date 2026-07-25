package handler

import (
	"growthos/internal/request"
	"growthos/internal/service"
	"growthos/pkg/response"
	"growthos/pkg/utils"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	UserService service.UserService
	JwtSecret   string
	JwtExpire   int
}

func NewAuthHandler(userService service.UserService, JwtSecret string, jwtExpire int) AuthHandler {
	return AuthHandler{
		UserService: userService,
		JwtSecret:   JwtSecret,
		JwtExpire:   jwtExpire,
	}
}

func (a *AuthHandler) Login(c *gin.Context) {
	var req request.LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "获取json失败")
		return
	}
	user, err := a.UserService.Login(req)
	if err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	token, err := utils.GenerateToken(user.Username, user.ID, a.JwtSecret, a.JwtExpire)
	if err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.Success(c, token)
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

func (a *AuthHandler) GetInfo(c *gin.Context) {
	id := c.GetUint64("user_id")
	user, err := a.UserService.Repo.FindUserById(id)
	if err != nil {
		response.Fail(c, 400, "获取信息失败")
		return
	}
	response.Success(c, user)
}
