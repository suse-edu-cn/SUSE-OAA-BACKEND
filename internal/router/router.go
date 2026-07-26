package router

import (
	"growthos/internal/handler"
	"growthos/internal/middleware"

	"github.com/gin-gonic/gin"
)

type Router struct {
	c         *gin.Engine
	JwtSecret string
	JwtExpire uint64
}

func RouterInit(total handler.TotalHandler) *gin.Engine {
	r := gin.Default()
	auth := r.Group("api/v2/auth")
	{
		auth.POST("login", total.Auth.Login)
		auth.POST("register", total.Auth.Register)
	}
	r.Use(middleware.JWTAuth(total.Auth.JwtSecret))
	user := r.Group("api/v2/users")
	{
		user.GET("/me", total.User.GetInfo)
	}
	return r
}
