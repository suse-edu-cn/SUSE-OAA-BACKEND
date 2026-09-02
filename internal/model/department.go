package model

import "time"

type Department struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	Name      string    `gorm:"type:varchar(50);not null;unique;column:name" json:"name"`
	Type      string    `gorm:"type:varchar(50);not null;column:type" json:"type"`
	IsActive  bool      `gorm:"not null;default:true;index;column:is_active" json:"is_active"`
	CreatedAt time.Time `gorm:"not null;column:created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;column:updated_at" json:"updated_at"`
}

var DefaultDepartments = []Department{
	{ID: 1, Name: "算法竞赛部", Type: "部门"},
	{ID: 2, Name: "组织宣传部", Type: "部门"},
	{ID: 3, Name: "秘书处", Type: "部门"},
	{ID: 4, Name: "理事会", Type: "部门"},
	{ID: 5, Name: "项目实践部", Type: "部门"},
	{ID: 6, Name: "开放原子开源协会", Type: "协会"},
}

func (Department) TableName() string { return "departments" }
