package model

import "time"

type Role struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	Name      string    `gorm:"type:varchar(50);not null;unique;column:name" json:"name"`
	Level     int       `gorm:"not null;column:level" json:"level"`
	CreatedAt time.Time `gorm:"not null;column:created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;column:updated_at" json:"updated_at"`
}

func (Role) TableName() string { return "roles" }
