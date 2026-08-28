package main

import (
	"suseoaa/internal/config"
	"suseoaa/internal/database"
	"suseoaa/internal/handler"
	"suseoaa/internal/repository"
	"suseoaa/internal/router"
	"suseoaa/internal/service"
)

func main() {
	Config := config.ConfigInit()
	db := database.MysqlInit(Config.Mysql)
	rdb := database.RedisInit(Config.Redis)

	repo := repository.NewUserRepository(db, rdb)
	roleRepo := repository.NewRoleRepository(db)
	departmentRepo := repository.NewDepartmentRepository(db)
	announcementRepo := repository.NewAnnouncementRepository(db)
	termRepo := repository.NewTermRepository(db)

	emailService := service.NewEmailService(Config.Email.Host, Config.Email.Port, Config.Email.User, Config.Email.Pass, Config.Email.Expire, Config.Email.CoolDown)
	userService := service.NewUserService(repo, roleRepo, departmentRepo, emailService)
	departmentService := service.NewDepartmentService(departmentRepo, roleRepo)
	roleService := service.NewRoleService(roleRepo)
	announcementService := service.NewAnnouncementService(announcementRepo, departmentRepo, roleRepo, repo)
	termService := service.NewTermService(termRepo, userService)

	userHandler := handler.NewUserHandler(userService)
	authHandler := handler.NewAuthHandler(userService, Config.Jwt.Secret, Config.Jwt.ExpireHour, Config.Jwt.RefreshTime)
	departmentHandler := handler.NewDepartmentHandler(departmentService)
	roleHandler := handler.NewRoleHandler(roleService)
	announcementHandler := handler.NewAnnouncementHandler(announcementService)
	termHandler := handler.NewTermHandler(termService)

	totalHandler := handler.NewTotalHandler(authHandler, userHandler, departmentHandler, roleHandler, announcementHandler, termHandler)

	r := router.RouterInit(totalHandler)
	r.Run(":" + Config.Server.Port)
}
