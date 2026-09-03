package model

import (
	"time"

	"gorm.io/plugin/soft_delete"
)

type User struct {
	ID           uint64                `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	StudentID    string                `gorm:"type:varchar(32);not null;column:student_id;uniqueIndex:idx_user_student_id" json:"student_id"`
	Username     string                `gorm:"type:varchar(64);not null;column:username;uniqueIndex:idx_user_username" json:"username"`
	Name         string                `gorm:"type:varchar(64);not null;column:name" json:"name"`
	Email        string                `gorm:"type:varchar(128);column:email;uniqueIndex:idx_user_email;default:null" json:"email"`
	Password     string                `gorm:"type:varchar(255);not null;column:password" json:"-"`
	DepartmentID uint64                `gorm:"index;column:department_id;not null" json:"department_id,omitempty"`
	Department   *Department           `gorm:"foreignKey:DepartmentID;references:ID" json:"department,omitempty"`
	RoleID       uint64                `gorm:"index;column:role_id;not null" json:"role_id,omitempty"`
	Role         *Role                 `gorm:"foreignKey:RoleID;references:ID" json:"role,omitempty"`
	Avatar       string                `gorm:"type:varchar(255);default:'avatar/default.png';column:avatar" json:"avatar"`
	CreatedAt    time.Time             `gorm:"not null;column:created_at" json:"created_at"`
	UpdatedAt    time.Time             `gorm:"not null;column:updated_at" json:"updated_at"`
	DeletedAt    soft_delete.DeletedAt `gorm:"softDelete:milli;uniqueIndex:idx_user_student_id;uniqueIndex:idx_user_username;uniqueIndex:idx_user_email" json:"-"`
}

type UserInfo struct {
	ID         uint64 `json:"id"`
	StudentID  string `json:"student_id"`
	Username   string `json:"username"`
	Name       string `json:"name"`
	Avatar     string `json:"avatar"`
	Email      string `json:"email"`
	Department string `json:"department"`
	Role       string `json:"role"`
}

type BatchUserInfo struct {
	ID           uint64 `json:"id"`
	StudentID    string `json:"student_id"`
	Username     string `json:"username"`
	Name         string `json:"name"`
	ErrorMessage string `json:"error_message"`
}
type UpdateUserItems struct {
	UserID       uint64  `json:"user_id"`
	DepartmentID *uint64 `json:"department_id"`
	RoleID       *uint64 `json:"role_id"`
}

func (User) TableName() string            { return "users" }
func (UserInfo) TableName() string        { return "users" }
func (BatchUserInfo) TableName() string   { return "users" }
func (UpdateUserItems) TableName() string { return "users" }
