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
	userService := service.NewUserService(repo)
	authHandler := handler.NewAuthHandler(userService, Config.Jwt.Secret, Config.Jwt.ExpireHour)
	r := router.RouterInit(authHandler)
	r.Run(":" + Config.Server.Port)
}
