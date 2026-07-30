package main

import (
	"growthos/internal/config"
	"growthos/internal/database"
	"growthos/internal/handler"
	"growthos/internal/repository"
	"growthos/internal/router"
	"growthos/internal/service"
)

func main() {
	Config := config.ConfigInit()
	db := database.MysqlInit(Config.Mysql)
	rdb := database.RedisInit(Config.Redis)

	repo := repository.NewUserRepository(db, rdb)
	roleRepo := repository.NewRoleRepository(db)
	departmentRepo := repository.NewDepartmentRepository(db)

	emailService := service.NewEmailService(Config.Email.Host, Config.Email.Port, Config.Email.User, Config.Email.Pass, Config.Email.Expire, Config.Email.CoolDown)
	userService := service.NewUserService(repo, roleRepo, departmentRepo, emailService)
	departmentService := service.NewDepartmentService(departmentRepo)
	roleService := service.NewRoleService(roleRepo)

	userHandler := handler.NewUserHandler(userService)
	authHandler := handler.NewAuthHandler(userService, Config.Jwt.Secret, Config.Jwt.ExpireHour, Config.Jwt.RefreshTime)
	departmentHandler := handler.NewDepartmentHandler(departmentService)
	roleHandler := handler.NewRoleHandler(roleService)

	totalHandler := handler.NewTotalHandler(authHandler, userHandler, departmentHandler, roleHandler)

	r := router.RouterInit(totalHandler)
	r.Run(":" + Config.Server.Port)
}
