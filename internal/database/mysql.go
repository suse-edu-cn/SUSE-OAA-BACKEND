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
	err = db.AutoMigrate(&model.User{}, &model.Department{}, &model.RefreshToken{}, &model.Role{}, &model.Announcement{})
	if err != nil {
		panic(err)
	}
	InitData(db)
	return db
}

func InitData(db *gorm.DB) {
	log.Println("开始初始化基础数据...")

	roles := []model.Role{
		{Name: "开发者", Level: 100},
		{Name: "会长", Level: 90},
		{Name: "副会长", Level: 80},
		{Name: "部长", Level: 60},
		{Name: "副部长", Level: 50},
		{Name: "干事", Level: 20},
		{Name: "会员", Level: 10},
	}

	roleMap := make(map[string]uint64)
	for _, r := range roles {
		role := r
		if err := db.Where(model.Role{Name: role.Name}).FirstOrCreate(&role).Error; err != nil {
			log.Printf("初始化角色 [%s] 失败: %v\n", role.Name, err)
		} else {
			roleMap[role.Name] = role.ID
		}
	}

	departments := []model.Department{
		{Name: "算法竞赛部"},
		{Name: "组织宣传部"},
		{Name: "秘书处"},
		{Name: "理事会"},
		{Name: "项目部"},
		{Name: "开放原子开源协会"},
	}

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
