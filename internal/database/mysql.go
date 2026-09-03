package database

import (
	"fmt"
	"log"
	"suseoaa/internal/config"
	"suseoaa/internal/model"

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
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	err = db.AutoMigrate(&model.User{},
		&model.Department{},
		&model.RefreshToken{},
		&model.Role{},
		&model.Announcement{},
		&model.Application{},
		&model.Interviewer{},
		&model.Term{},
		&model.TermHistory{},
		&model.InterviewResult{})
	if err != nil {
		panic(err)
	}
	InitData(db)
	return db
}

func InitData(db *gorm.DB) {
	log.Println("开始初始化基础数据...")

	roles := model.DefaultRoles

	roleMap := make(map[string]uint64)
	for _, r := range roles {
		role := r
		if err := db.Where(model.Role{Name: role.Name}).FirstOrCreate(&role).Error; err != nil {
			log.Printf("初始化角色 [%s] 失败: %v\n", role.Name, err)
		} else {
			roleMap[role.Name] = role.ID
		}
	}

	departments := model.DefaultDepartments
	deptMap := make(map[string]uint64)
	for _, d := range departments {
		dept := d
		if err := db.Where(model.Department{Name: dept.Name}).FirstOrCreate(&dept).Error; err != nil {
			log.Printf("初始化部门 [%s] 失败: %v\n", dept.Name, err)
		} else {
			deptMap[dept.Name] = dept.ID
		}
	}

	log.Println(" 基础数据初始化完成！")
}
