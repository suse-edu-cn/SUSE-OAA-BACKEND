package router

import (
	"time"

	"suseoaa/internal/handler"
	"suseoaa/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type Router struct {
	c         *gin.Engine
	JwtSecret string
	JwtExpire uint64
}

func RouterInit(total *handler.TotalHandler, rdb *redis.Client, trustedProxies []string) *gin.Engine {
	r := gin.New()

	if len(trustedProxies) > 0 {
		_ = r.SetTrustedProxies(trustedProxies)
	} else {
		_ = r.SetTrustedProxies([]string{"127.0.0.1", "::1"})
	}

	r.Use(middleware.Trace())
	r.Use(middleware.AccessLog())
	r.Use(gin.Recovery())
	r.Use(middleware.CORS())

	r.GET("/livez", total.Health.Livez)
	r.GET("/healthz", total.Health.Healthz)
	r.GET("/.well-known/jwks.json", total.OAuth.JWKS)
	r.GET("/.well-known/openid-configuration", total.OAuth.OpenIDConfiguration)

	const apiPrefix = "v2"

	api := r.Group(apiPrefix)
	{
		api.GET("livez", total.Health.Livez)
		api.GET("healthz", total.Health.Healthz)

		loginLimiter := middleware.RateLimit(rdb, "login", 10, time.Minute)
		sendLimiter := middleware.RateLimit(rdb, "send_code", 5, time.Minute)
		resetLimiter := middleware.RateLimit(rdb, "reset_password", 5, time.Minute)

		auth := api.Group("auth")
		{
			auth.POST("login", loginLimiter, total.Auth.Login)
			auth.POST("register", total.Auth.Register)
			auth.POST("refresh", total.Auth.Refresh)
			auth.POST("password/reset", resetLimiter, total.Auth.ResetPassword)
			auth.POST("send", sendLimiter, total.Auth.SendVerificationCode)
			auth.Use(middleware.JWTAuth(total.Auth.JwtSecret))
			auth.POST("logout", total.Auth.Logout)
		}

		password := api.Group("auth/password", middleware.JWTAuth(total.Auth.JwtSecret))
		{
			password.POST("update", total.Auth.UpdatePassword)
		}

		user := api.Group("user", middleware.JWTAuth(total.Auth.JwtSecret))
		{
			user.GET("me", total.User.GetInfo)
			user.GET("list", total.User.GetUserList)
			user.POST("me/update", total.User.UpdateUserInfo)
			user.POST("batch", total.User.BatchUserInfo)
			user.POST("delete", total.User.DeleteUser)
			user.POST("delete/cancel", total.User.CancelDeleteUser)
		}

		department := api.Group("department", middleware.JWTAuth(total.Auth.JwtSecret))
		{
			department.GET("list", total.Department.GetAll)
			department.POST("create", total.Department.Create)
			department.POST("update", total.Department.Update)
		}

		role := api.Group("role", middleware.JWTAuth(total.Auth.JwtSecret))
		{
			role.GET("list", total.Role.FindAll)
			role.POST("create", total.Role.Create)
			role.POST("update", total.Role.Update)
		}

		announcement := api.Group("announcement", middleware.JWTAuth(total.Auth.JwtSecret))
		{
			announcement.POST("create", total.Announcement.CreateAnnouncement)
			announcement.POST("update", total.Announcement.UpdateAnnouncement)
			announcement.POST("push", total.Announcement.PushAnnouncement)
			announcement.GET("list", total.Announcement.GetAnnouncementList)
			announcement.GET("get", total.Announcement.GetAnnouncement)
			announcement.POST("delete", total.Announcement.DeleteAnnouncement)
		}

		term := api.Group("term", middleware.JWTAuth(total.Auth.JwtSecret))
		{
			term.POST("create", total.Term.CreateTerm)
			term.POST("update", total.Term.UpdateTerm)
			term.GET("list", total.Term.GetTermList)
			term.POST("delete", total.Term.DeleteTerm)
		}

		application := api.Group("application", middleware.JWTAuth(total.Auth.JwtSecret))
		{
			application.POST("create", total.Term.CreateApplication)
			application.GET("department", total.Term.GetApplicationDepartmentList)
			application.GET("role", total.Term.GetApplicationRoleList)
			application.POST("update", total.Term.UpdateApplication)
			application.GET("me", total.Term.GetMyApplications)
			application.GET("list", total.Term.GetApplicationList)
			application.POST("delete", total.Term.DeleteApplication)
		}

		interviewer := api.Group("interviewer", middleware.JWTAuth(total.Auth.JwtSecret))
		{
			interviewer.POST("create", total.Term.CreateInterviewers)
			interviewer.POST("update", total.Term.UpdateInterviewer)
			interviewer.GET("list", total.Term.GetInterviewerList)
			interviewer.POST("delete", total.Term.DeleteInterviewer)

			result := interviewer.Group("result", middleware.JWTAuth(total.Auth.JwtSecret))
			{
				result.POST("create", total.Term.CreateInterviewResult)
				result.POST("update", total.Term.UpdateInterviewResult)
				result.POST("delete", total.Term.DeleteInterviewResult)
				result.GET("list", total.Term.GetInterviewResultList)
				result.GET("decision", total.Term.GetInterviewResultDecision)
				result.GET("export", total.Term.ExportInterviewResults)
				result.GET("fields", total.Term.GetExportFields)
			}
		}

		upload := api.Group("upload", middleware.JWTAuth(total.Auth.JwtSecret))
		{
			upload.POST("image", total.File.UploadImage)
			upload.POST("file", total.File.UploadFile)
		}

		app := api.Group("app", middleware.JWTAuth(total.Auth.JwtSecret))
		{
			app.POST("create", total.OAuth.CreateOAuthClient)
			app.GET("detail", total.OAuth.GetOAuthClient)
			app.POST("update", total.OAuth.UpdateOAuthClient)
			app.GET("list", total.OAuth.GetOAuthClientList)
			app.POST("delete", total.OAuth.DeleteOAuthClient)
			app.POST("secret/reset", total.OAuth.ResetOAuthClientSecret)
		}

		oAuth := api.Group("oauth")
		{
			oAuth.GET("authorize", total.OAuth.Authorize)
			oAuth.POST("token", total.OAuth.Token)
			oAuth.GET("userinfo", total.OAuth.UserInfo)
			oAuth.POST("userinfo", total.OAuth.UserInfo)
			oAuth.GET("client/public", total.OAuth.GetPublicClientInfo)
		}
	}
	return r
}
