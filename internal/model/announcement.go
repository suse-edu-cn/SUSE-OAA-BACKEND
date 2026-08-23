package model

import "time"

type Announcement struct {
	ID           uint64      `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	Title        string      `gorm:"type:varchar(255);not null;column:title" json:"title"`
	Content      string      `gorm:"type:mediumtext;not null;column:content" json:"content"`
	IsActive     bool        `gorm:"not null;default:false;column:is_active" json:"is_active"`
	DepartmentID uint64      `gorm:"index;not null;column:department_id" json:"department_id"`
	Department   *Department `gorm:"foreignKey:DepartmentID;references:ID" json:"department,omitempty"`
	PublisherID  *uint64     `gorm:"index;column:publisher_id" json:"publisher_id"`
	Publisher    *User       `gorm:"foreignKey:PublisherID;references:ID" json:"publisher,omitempty"`
	PublishedAt  *time.Time  `gorm:"column:published_at" json:"published_at,omitempty"`
	CreatedAt    time.Time   `gorm:"not null;column:created_at" json:"created_at"`
	UpdatedAt    time.Time   `gorm:"not null;column:updated_at" json:"updated_at"`
}

type AnnouncementInfo struct {
	ID             uint64     `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	Title          string     `gorm:"type:varchar(255);not null;column:title" json:"title"`
	Content        string     `gorm:"type:mediumtext;not null;column:content" json:"content"`
	IsActive       bool       `gorm:"not null;default:false;column:is_active" json:"is_active"`
	DepartmentName string     `gorm:"type:varchar(255);not null;column:department_name" json:"department_name"`
	PublisherID    *uint64    `gorm:"index;column:publisher_id" json:"publisher_id"`
	PublisherName  string     `gorm:"type:varchar(255);not null;column:publisher_name" json:"publisher_name"`
	PublisherRole  string     `gorm:"type:varchar(255);not null;column:publisher_role" json:"publisher_role"`
	PublishedAt    *time.Time `gorm:"column:published_at" json:"published_at"`
	CreatedAt      time.Time  `gorm:"not null;column:created_at" json:"created_at"`
	UpdatedAt      time.Time  `gorm:"not null;column:updated_at" json:"updated_at"`
}

func (a *Announcement) TableName() string     { return "announcements" }
func (a *AnnouncementInfo) TableName() string { return "announcements" }

func (a *Announcement) ToInfo() AnnouncementInfo {
	if a == nil {
		return AnnouncementInfo{}
	}
	info := AnnouncementInfo{
		ID:             a.ID,
		Title:          a.Title,
		Content:        a.Content,
		IsActive:       a.IsActive,
		DepartmentName: a.Department.Name,
		CreatedAt:      a.CreatedAt,
		UpdatedAt:      a.UpdatedAt,
	}
	if a.Publisher != nil {
		info.PublisherName = a.Publisher.Name
		info.PublishedAt = a.PublishedAt
		info.PublisherID = a.PublisherID
		if a.Publisher.Role != nil {
			info.PublisherRole = a.Publisher.Role.Name
		}
	}
	return info
}
