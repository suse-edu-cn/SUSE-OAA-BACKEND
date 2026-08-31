package model

import "time"

type Role struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	Name      string    `gorm:"type:varchar(50);not null;unique;column:name" json:"name"`
	Level     uint64    `gorm:"not null;column:level" json:"level"`
	Type      string    `gorm:"type:varchar(50);not null;column:type" json:"type"`
	IsActive  bool      `gorm:"not null;default:true;index;column:is_active" json:"is_active"`
	CreatedAt time.Time `gorm:"not null;column:created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;column:updated_at" json:"updated_at"`
}

var DefaultRoles = []Role{
	{ID: 1, Name: "开发者", Level: 100, Type: "协会"},
	{ID: 2, Name: "会长", Level: 90, Type: "协会"},
	{ID: 3, Name: "副会长", Level: 80, Type: "协会"},
	{ID: 4, Name: "部长", Level: 60, Type: "部门"},
	{ID: 5, Name: "副部长", Level: 50, Type: "部门"},
	{ID: 6, Name: "干事", Level: 20, Type: "部门"},
	{ID: 7, Name: "会员", Level: 10, Type: "协会"},
}

func (Role) TableName() string { return "roles" }
