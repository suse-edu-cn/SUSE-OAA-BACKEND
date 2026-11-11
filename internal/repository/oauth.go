package repository

import (
	"context"
	"encoding/json"
	"suseoaa/internal/model"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type OAuthRepository struct {
	DB  *gorm.DB
	Rdb *redis.Client
}

func NewOAuthRepository(db *gorm.DB, rdb *redis.Client) *OAuthRepository {
	return &OAuthRepository{
		DB:  db,
		Rdb: rdb,
	}
}

func (o *OAuthRepository) CreateOAuthClient(ctx context.Context, client model.OAuthClient) error {
	return o.DB.WithContext(ctx).Create(&client).Error
}
func (o *OAuthRepository) UpdateOAuthClient(ctx context.Context, client map[string]any, clientID string) error {
	return o.DB.WithContext(ctx).Model(&model.OAuthClient{}).Where("client_id = ?", clientID).Updates(client).Error
}
func (o *OAuthRepository) GetOAuthClientInfoByClientID(ctx context.Context, clientID string) (model.OAuthClientInfo, error) {
	var client model.OAuthClient
	err := o.DB.WithContext(ctx).Model(&model.OAuthClient{}).Where("client_id = ?", clientID).First(&client).Error
	if err != nil {
		return model.OAuthClientInfo{}, err
	}
	return client.ToInfo(), nil
}
func (o *OAuthRepository) GetOAuthClientByClientID(ctx context.Context, clientID string) (*model.OAuthClient, error) {
	var client model.OAuthClient
	err := o.DB.WithContext(ctx).Model(&model.OAuthClient{}).Where("client_id = ?", clientID).First(&client).Error
	if err != nil {
		return nil, err
	}
	return &client, nil
}
func (o *OAuthRepository) GetOAuthClientList(ctx context.Context) ([]model.OAuthClientInfo, error) {
	var clients []model.OAuthClient
	err := o.DB.WithContext(ctx).Model(&model.OAuthClient{}).Find(&clients).Error
	if err != nil {
		return nil, err
	}
	var clientInfos []model.OAuthClientInfo
	for _, client := range clients {
		clientInfos = append(clientInfos, client.ToInfo())
	}
	return clientInfos, nil
}
func (o *OAuthRepository) DeleteOAuthClient(ctx context.Context, clientID string) error {
	return o.DB.WithContext(ctx).Model(&model.OAuthClient{}).Where("client_id = ?", clientID).Delete(&model.OAuthClient{}).Error
}
func (o *OAuthRepository) ResetOAuthClientSecret(ctx context.Context, clientID string, secret string, maskSecret string) error {
	return o.DB.WithContext(ctx).Model(&model.OAuthClient{}).Where("client_id = ?", clientID).Updates(map[string]any{
		"client_secret": secret,
		"mask_secret":   maskSecret,
	}).Error
}
func (o *OAuthRepository) SaveAuthCode(ctx context.Context, code string, data model.AuthCodeData, ttl time.Duration) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	key := "oauth:code:" + code
	return o.Rdb.Set(ctx, key, bytes, ttl).Err()
}

func (o *OAuthRepository) GetAuthCodeByKey(ctx context.Context, key string) (string, error) {
	return o.Rdb.GetDel(ctx, key).Result()
}

func (o *OAuthRepository) SetToken(ctx context.Context, token string, tokenData any, expireTime time.Duration) error {
	return o.Rdb.Set(ctx, token, tokenData, expireTime).Err()
}
func (o *OAuthRepository) GetTokenByKey(ctx context.Context, key string) (string, error) {
	return o.Rdb.Get(ctx, key).Result()
}
