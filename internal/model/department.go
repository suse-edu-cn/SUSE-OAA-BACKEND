package model

import "time"

type Department struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	Name      string    `gorm:"type:varchar(50);not null;unique;column:name" json:"name"`
	Type      string    `gorm:"type:varchar(50);not null;column:type" json:"type"`
	CreatedAt time.Time `gorm:"not null;column:created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;column:updated_at" json:"updated_at"`
}

var DefaultDepartments = []Department{
	{Name: "算法竞赛部", Type: "部门"},
	{Name: "组织宣传部", Type: "部门"},
	{Name: "秘书处", Type: "部门"},
	{Name: "理事会", Type: "部门"},
	{Name: "项目部", Type: "部门"},
	{Name: "开放原子开源协会", Type: "协会"},
}

func (Department) TableName() string { return "departments" }
