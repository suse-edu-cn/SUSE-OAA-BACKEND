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

	userService := service.NewUserService(repo, roleRepo, departmentRepo)

	userHandler := handler.NewUserHandler(userService)
	authHandler := handler.NewAuthHandler(userService, Config.Jwt.Secret, Config.Jwt.ExpireHour)
	totalHadler := handler.NewTotalHandler(authHandler, userHandler)

	r := router.RouterInit(totalHadler)
	r.Run(":" + Config.Server.Port)
}
