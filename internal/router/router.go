package router

import (
	"suseoaa/internal/handler"
	"suseoaa/internal/middleware"

	"github.com/gin-gonic/gin"
)

type Router struct {
	c         *gin.Engine
	JwtSecret string
	JwtExpire uint64
}

func RouterInit(total handler.TotalHandler) *gin.Engine {
	r := gin.Default()
	r.Use(middleware.CORS())

	auth := r.Group("api/v2/auth")
	{
		auth.POST("login", total.Auth.Login)
		auth.POST("register", total.Auth.Register)
		auth.POST("refresh", total.Auth.Refresh)

		auth.Use(middleware.JWTAuth(total.Auth.JwtSecret))
		auth.POST("logout", total.Auth.Logout)
		auth.POST("send", total.Auth.SendVerificationCode)
	}
	r.Use(middleware.JWTAuth(total.Auth.JwtSecret))
	password := r.Group("api/v2/password")
	{
		password.POST("update", total.Auth.UpdatePassword)
		password.POST("reset", total.Auth.ResetPassword)
	}
	user := r.Group("api/v2/user")
	{
		user.GET("me", total.User.GetInfo)
		user.GET("list", total.User.GetUserList)
		user.POST("me/update", total.User.UpdateUserInfo)
		user.POST("batch", total.User.BatchUserInfo)
	}
	department := r.Group("api/v2/department")
	{
		department.GET("list", total.Department.GetAll)
	}
	role := r.Group("api/v2/role")
	{
		role.GET("list", total.Role.FindAll)
	}
	announcement := r.Group("api/v2/announcement")
	{
		announcement.POST("create", total.Announcement.CreateAnnouncement)
		announcement.POST("update", total.Announcement.UpdateAnnouncement)
		announcement.POST("push", total.Announcement.PushAnnouncement)
		announcement.GET("active", total.Announcement.GetAnnouncementActiveList)
		announcement.GET("history", total.Announcement.GetAnnouncementHistoryList)
	}
	return r
}
