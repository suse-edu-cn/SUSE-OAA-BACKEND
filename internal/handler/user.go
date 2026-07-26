package handler

import (
	"growthos/internal/service"
	"growthos/pkg/response"

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
	result, err := u.UserService.GetUserInfo(id)
	if err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.Success(c, result)
}
