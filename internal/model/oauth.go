package model

import (
	"time"

	"gorm.io/plugin/soft_delete"
)

type OAuthClient struct {
	ID           string                `gorm:"type:varchar(36);primaryKey;column:id" json:"id"`
	ClientID     string                `gorm:"type:varchar(64);not null;uniqueIndex:idx_client_id;column:client_id" json:"client_id"`
	ClientSecret string                `gorm:"type:varchar(255);not null;column:client_secret" json:"-"`
	MaskSecret   string                `gorm:"type:varchar(255);not null;column:mask_secret" json:"-"`
	Name         string                `gorm:"type:varchar(64);not null;column:name" json:"name"`
	RedirectURIs string                `gorm:"type:text;not null;column:redirect_uris" json:"redirect_uris"`
	Scope        string                `gorm:"type:varchar(128);default:'openid profile email';column:scope" json:"scope"`
	IsActive     bool                  `gorm:"not null;default:true;column:is_active" json:"is_active"`
	CreatedAt    time.Time             `gorm:"not null;column:created_at" json:"created_at"`
	UpdatedAt    time.Time             `gorm:"not null;column:updated_at" json:"updated_at"`
	DeletedAt    soft_delete.DeletedAt `gorm:"softDelete:milli;uniqueIndex:idx_client_id" json:"-"`
}

type OAuthClientInfo struct {
	ClientID     string    `json:"client_id"`
	MaskSecret   string    `json:"mask_secret"`
	Name         string    `json:"name"`
	RedirectURIs string    `json:"redirect_uris"`
	Scope        string    `json:"scope"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
type AuthCodeData struct {
	UserID      uint64 `json:"user_id"`
	ClientID    string `json:"client_id"`
	RedirectURI string `json:"redirect_uri"`
	Scope       string `json:"scope"`
}

func (o *OAuthClient) TableName() string {
	return "oauth_clients"
}
func (o *OAuthClient) ToInfo() OAuthClientInfo {
	return OAuthClientInfo{
		ClientID:     o.ClientID,
		MaskSecret:   o.MaskSecret,
		Name:         o.Name,
		RedirectURIs: o.RedirectURIs,
		Scope:        o.Scope,
		IsActive:     o.IsActive,
		CreatedAt:    o.CreatedAt,
		UpdatedAt:    o.UpdatedAt,
	}
}
