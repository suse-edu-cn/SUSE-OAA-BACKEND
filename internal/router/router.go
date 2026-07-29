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
		auth.POST("refresh", total.Auth.Refresh)
	}
	r.Use(middleware.JWTAuth(total.Auth.JwtSecret))
	user := r.Group("api/v2/user")
	{
		user.GET("/me", total.User.GetInfo)
	}
	department := r.Group("api/v2/department")
	{
		department.GET("/list", total.Department.GetAll)
	}
	role := r.Group("api/v2/role")
	{
		role.GET("/list", total.Role.FindAll)
	}
	return r
}
