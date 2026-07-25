package model

import "time"

type User struct {
	ID           uint64      `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	StudentID    string      `gorm:"type:varchar(32);not null;unique;column:student_id" json:"student_id"`
	Username     string      `gorm:"type:varchar(64);not null;column:username" json:"username"`
	Name         string      `gorm:"type:varchar(64);not null;column:name" json:"name"`
	Email        string      `gorm:"type:varchar(128);not null;unique;column:email" json:"email"`
	Password     string      `gorm:"type:varchar(255);not null;column:password" json:"-"` // 默认不进行 JSON 序列化
	DepartmentID *uint64     `gorm:"index;column:department_id" json:"department_id,omitempty"`
	Department   *Department `gorm:"foreignKey:DepartmentID;references:ID" json:"department,omitempty"`
	RoleID       *uint64     `gorm:"index;column:role_id" json:"role_id,omitempty"`
	Role         *Role       `gorm:"foreignKey:RoleID;references:ID" json:"role,omitempty"`
	Avatar       string      `gorm:"type:varchar(255);default:'images/default.jpg';column:avatar" json:"avatar"`
	CreatedAt    time.Time   `gorm:"not null;column:created_at" json:"created_at"`
	UpdatedAt    time.Time   `gorm:"not null;column:updated_at" json:"updated_at"`
}

func (User) TableName() string { return "user" }
