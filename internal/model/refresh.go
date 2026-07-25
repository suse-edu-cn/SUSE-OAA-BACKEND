package model

import "time"

type RefreshToken struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	UserID    uint64    `gorm:"not null;index;column:user_id" json:"user_id"`
	User      *User     `gorm:"foreignKey:UserID;references:ID" json:"user,omitempty"`
	Token     string    `gorm:"type:varchar(255);not null;unique;column:token" json:"token"`
	Device    string    `gorm:"type:varchar(20);not null;column:device" json:"device"`
	CreatedAt time.Time `gorm:"not null;column:created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;column:updated_at" json:"updated_at"`
}

func (RefreshToken) TableName() string { return "refresh_tokens" }
