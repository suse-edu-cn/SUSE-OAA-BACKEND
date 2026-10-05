package main

import (
	"context"
	"errors"
	"log/slog"
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
	"suseoaa/pkg/logger"
	"sync"
	"syscall"
	"time"
)

func main() {
	logger.Init(slog.LevelInfo)

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
		logger.Info("HTTP服务正在启动", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("HTTP服务异常退出", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("接收到退出信号，启动优雅停机流程")

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP 服务强制关闭", "err", err)
	} else {
		logger.Info("HTTP 服务已完成存量请求处理并关闭")
	}
	cancelWorkers()
	wg.Wait()
	logger.Info("所有后台常驻Worker已退出")

	logger.Info("服务已安全下线")
}
