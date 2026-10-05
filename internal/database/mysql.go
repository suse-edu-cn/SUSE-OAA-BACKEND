package database

import (
	"fmt"
	"time"

	"suseoaa/internal/config"
	"suseoaa/internal/model"
	"suseoaa/pkg/logger"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func MysqlInit(cfg config.Mysql) *gorm.DB {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.Username,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Database,
	)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		panic(err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		panic(fmt.Sprintf("获取底层 sql.DB 失败: %v", err))
	}
	maxOpen := cfg.MaxOpenConns
	if maxOpen <= 0 {
		maxOpen = 100 // 默认兜底
	}
	maxIdle := cfg.MaxIdleConns
	if maxIdle <= 0 {
		maxIdle = 25 // 默认兜底
	}
	lifetime := time.Duration(cfg.ConnMaxLifetimeMinute) * time.Minute
	if lifetime <= 0 {
		lifetime = time.Hour
	}
	idleTime := time.Duration(cfg.ConnMaxIdleTimeMinute) * time.Minute
	if idleTime <= 0 {
		idleTime = 10 * time.Minute
	}
	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetMaxIdleConns(maxIdle)
	sqlDB.SetConnMaxLifetime(lifetime)
	sqlDB.SetConnMaxIdleTime(idleTime)
	err = db.AutoMigrate(&model.User{},
		&model.Department{},
		&model.RefreshToken{},
		&model.Role{},
		&model.Announcement{},
		&model.Application{},
		&model.Interviewer{},
		&model.Term{},
		&model.InterviewResult{})
	if err != nil {
		panic(err)
	}
	InitData(db)
	return db
}

func InitData(db *gorm.DB) {
	logger.Info("开始初始化基础数据...")

	roles := model.DefaultRoles

	roleMap := make(map[string]uint64)
	for _, r := range roles {
		role := r
		if err := db.Where(model.Role{Name: role.Name}).FirstOrCreate(&role).Error; err != nil {
			logger.Warn("初始化角色失败", "role", role.Name, "err", err)
		} else {
			roleMap[role.Name] = role.ID
		}
	}

	departments := model.DefaultDepartments
	deptMap := make(map[string]uint64)
	for _, d := range departments {
		dept := d
		if err := db.Where(model.Department{Name: dept.Name}).FirstOrCreate(&dept).Error; err != nil {
			logger.Warn("初始化部门失败", "dept", dept.Name, "err", err)
		} else {
			deptMap[dept.Name] = dept.ID
		}
	}

	logger.Info("基础数据初始化完成！")
}
