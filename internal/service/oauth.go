package service

import (
	"context"
	"errors"
	"suseoaa/internal/model"
	"suseoaa/internal/repository"
	"suseoaa/internal/request"
	"suseoaa/internal/storage"
	"suseoaa/pkg/utils"
	"uuid"

	"golang.org/x/crypto/bcrypt"
)

type OAuthService struct {
	UserRepo  *repository.UserRepository
	OAuthRepo *repository.OAuthRepository
	ImgMinio  *storage.MinIO
}

func NewOAuthService(userRepo *repository.UserRepository, oauthRepo *repository.OAuthRepository, imgMinio *storage.MinIO) *OAuthService {
	return &OAuthService{
		UserRepo:  userRepo,
		OAuthRepo: oauthRepo,
		ImgMinio:  imgMinio,
	}
}
func (o *OAuthService) CheckLevel(ctx context.Context, userID uint64) error {
	level, _, err := o.UserRepo.GetRoleLevelAndDepartment(ctx, userID)
	if err != nil {
		return err
	}
	if level < 80 {
		return errors.New("权利不够")
	}
	return nil
}
func (o *OAuthService) CreateOAuthClient(ctx context.Context, req request.CreateOAuthClientReq, userID uint64) (map[string]string, error) {
	err := o.CheckLevel(ctx, userID)
	if err != nil {
		return nil, err
	}
	secret, maskSecret, err := utils.GenerateClientSecret()
	if err != nil {
		return nil, err
	}
	hashSecret, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	client := model.OAuthClient{
		ID:           uuid.NewV7().String(),
		ClientID:     req.ClientID,
		ClientSecret: string(hashSecret),
		MaskSecret:   maskSecret,
		Name:         req.Name,
		RedirectURIs: req.RedirectURIs,
		Scope:        req.Scope,
		IsActive:     true,
	}
	err = o.OAuthRepo.CreateOAuthClient(ctx, client)
	if err != nil {
		return nil, err
	}
	return map[string]string{
		"client_id":     client.ClientID,
		"client_secret": secret,
		"name":          client.Name,
	}, nil
}
func (o *OAuthService) UpdateOAuthClient(ctx context.Context, userID uint64, req request.UpdateOAuthClientReq) error {
	err := o.CheckLevel(ctx, userID)
	if err != nil {
		return err
	}
	_, err = o.OAuthRepo.GetOAuthClientByClientID(ctx, req.ClientID)
	if err != nil {
		return errors.New("应用不存在")
	}
	res := map[string]any{
		"name":          req.Name,
		"redirect_uris": req.RedirectURIs,
		"scope":         req.Scope,
	}
	if req.IsActive != nil {
		res["is_active"] = *req.IsActive
	}
	return o.OAuthRepo.UpdateOAuthClient(ctx, res, req.ClientID)

}
func (o *OAuthService) GetOAuthClientByID(ctx context.Context, clientID string, userID uint64) (model.OAuthClientInfo, error) {
	err := o.CheckLevel(ctx, userID)
	if err != nil {
		return model.OAuthClientInfo{}, err
	}
	res, err := o.OAuthRepo.GetOAuthClientByClientID(ctx, clientID)
	if err != nil {
		return model.OAuthClientInfo{}, errors.New("应用不存在")
	}
	return res, nil
}
func (o *OAuthService) GetOAuthClientList(ctx context.Context, userID uint64) ([]model.OAuthClientInfo, error) {
	err := o.CheckLevel(ctx, userID)
	if err != nil {
		return nil, err
	}
	return o.OAuthRepo.GetOAuthClientList(ctx)
}
func (o *OAuthService) DeleteOAuthClient(ctx context.Context, clientID string, userID uint64) error {
	err := o.CheckLevel(ctx, userID)
	if err != nil {
		return err
	}
	_, err = o.OAuthRepo.GetOAuthClientByClientID(ctx, clientID)
	if err != nil {
		return errors.New("应用不存在")
	}
	return o.OAuthRepo.DeleteOAuthClient(ctx, clientID)
}
