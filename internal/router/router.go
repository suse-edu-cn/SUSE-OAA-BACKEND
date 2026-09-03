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

	auth := r.Group("v2/auth")
	{
		auth.POST("login", total.Auth.Login)
		auth.POST("register", total.Auth.Register)
		auth.POST("refresh", total.Auth.Refresh)
		auth.POST("password/reset", total.Auth.ResetPassword)
		auth.POST("send", total.Auth.SendVerificationCode)
		auth.Use(middleware.JWTAuth(total.Auth.JwtSecret))
		auth.POST("logout", total.Auth.Logout)

	}
	r.Use(middleware.JWTAuth(total.Auth.JwtSecret))
	password := r.Group("v2/auth/password")
	{
		password.POST("update", total.Auth.UpdatePassword)
	}
	user := r.Group("v2/user")
	{
		user.GET("me", total.User.GetInfo)
		user.GET("list", total.User.GetUserList)
		user.POST("me/update", total.User.UpdateUserInfo)
		user.POST("batch", total.User.BatchUserInfo)
		user.POST("delete", total.User.DeleteUser)
	}
	department := r.Group("v2/department")
	{
		department.GET("list", total.Department.GetAll)
		department.POST("create", total.Department.Create)
		department.POST("update", total.Department.Update)
	}
	role := r.Group("v2/role")
	{
		role.GET("list", total.Role.FindAll)
		role.POST("create", total.Role.Create)
		role.POST("update", total.Role.Update)
	}
	announcement := r.Group("v2/announcement")
	{
		announcement.POST("create", total.Announcement.CreateAnnouncement)
		announcement.POST("update", total.Announcement.UpdateAnnouncement)
		announcement.POST("push", total.Announcement.PushAnnouncement)
		announcement.GET("list", total.Announcement.GetAnnouncementList)
		announcement.POST("delete", total.Announcement.DeleteAnnouncement)
	}
	term := r.Group("v2/term")
	{
		term.POST("create", total.Term.CreateTerm)
		term.POST("update", total.Term.UpdateTerm)
		term.GET("list", total.Term.GetTermList)
	}
	application := r.Group("v2/application")
	{
		application.POST("create", total.Term.CreateApplication)
		application.GET("department", total.Term.GetApplicationDepartmentList)
		application.GET("role", total.Term.GetApplicationRoleList)
		application.POST("update", total.Term.UpdateApplication)
		application.GET("me", total.Term.GetMyApplications)
		application.GET("list", total.Term.GetApplicationList)
		application.POST("delete", total.Term.DeleteApplication)
	}
	interviewer := r.Group("v2/interviewer")
	{
		interviewer.POST("create", total.Term.CreateInterviewers)
		interviewer.POST("update", total.Term.UpdateInterviewer)
		interviewer.GET("list", total.Term.GetInterviewerList)

		result := interviewer.Group("result")
		{
			result.POST("create", total.Term.CreateInterviewResult)
			result.POST("update", total.Term.UpdateInterviewResult)
			result.GET("decision", total.Term.GetInterviewResultDecision)
		}
	}
	upload := r.Group("v2/upload")
	{
		upload.POST("image", total.File.UploadImage)
	}
	return r
}
