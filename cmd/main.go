package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"suseoaa/internal/config"
	"suseoaa/internal/database"
	"suseoaa/internal/handler"
	"suseoaa/internal/repository"
	"suseoaa/internal/router"
	"suseoaa/internal/service"
	"suseoaa/internal/storage"
	"sync"
	"syscall"
	"time"
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
	imgMinio, fileMinio := storage.NewMinIO(
		Config.MiniO.MinioEndpoint,
		Config.MiniO.PublicEndpoint,
		Config.MiniO.MinioRegion,
		Config.MiniO.MinioAccessKey,
		Config.MiniO.MinioSecretKey,
		Config.MiniO.MinioUseSsl,
		Config.MiniO.PublicUseSsl,
		Config.MiniO.MinioImgBucket,
		Config.MiniO.MinioFileBucket,
		Config.MiniO.MaxFileSize,
		Config.MiniO.MaxImageSize,
		Config.MiniO.ExpireTime)

	emailService := service.NewEmailService(Config.Email.Host,
		Config.Email.Port,
		Config.Email.User,
		Config.Email.Pass,
		Config.Email.Expire,
		Config.Email.CoolDown)
	fileService := service.NewFileService(imgMinio, fileMinio)
	userService := service.NewUserService(repo, roleRepo, departmentRepo, emailService, fileService)
	departmentService := service.NewDepartmentService(departmentRepo, roleRepo)
	roleService := service.NewRoleService(roleRepo)
	announcementService := service.NewAnnouncementService(announcementRepo, departmentRepo, roleRepo, repo, fileService)
	termService := service.NewTermService(termRepo, userService)

	workerCtx, cancelWorkers := context.WithCancel(context.Background())
	defer cancelWorkers()

	var wg sync.WaitGroup

	wg.Go(func() {
		termService.StartInterviewResultExecutor(workerCtx)
	})
	wg.Go(func() {
		userService.StartUserDeletionExecutor(workerCtx)
	})

	userHandler := handler.NewUserHandler(userService)
	authHandler := handler.NewAuthHandler(
		userService,
		Config.Jwt.Secret,
		Config.Jwt.ExpireMinute,
		Config.Jwt.RefreshTime)
	departmentHandler := handler.NewDepartmentHandler(departmentService)
	roleHandler := handler.NewRoleHandler(roleService)
	announcementHandler := handler.NewAnnouncementHandler(announcementService)
	termHandler := handler.NewTermHandler(termService)
	fileHandler := handler.NewFileHandler(fileService)

	totalHandler := handler.NewTotalHandler(
		authHandler,
		userHandler,
		departmentHandler,
		roleHandler,
		announcementHandler,
		termHandler,
		fileHandler)

	r := router.RouterInit(totalHandler)

	srv := &http.Server{
		Addr:    Config.Server.Host + ":" + Config.Server.Port,
		Handler: r,
	}

	go func() {
		log.Printf("HTTP服务正在启动，监听地址: %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP服务异常退出: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("接收到退出信号")

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP 服务强制关闭: %v", err)
	} else {
		log.Println("HTTP 服务已关闭")
	}
	cancelWorkers()
	wg.Wait()
	log.Println("所有后台常驻Worker已退出")

	log.Println("服务已下线")
}
