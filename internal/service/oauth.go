package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"suseoaa/internal/model"
	"suseoaa/internal/repository"
	"suseoaa/internal/request"
	"suseoaa/internal/storage"
	"suseoaa/pkg/utils"
	"time"
	"uuid"

	"golang.org/x/crypto/bcrypt"
)

type OAuthService struct {
	UserRepo    *repository.UserRepository
	OAuthRepo   *repository.OAuthRepository
	ImgMinio    *storage.MinIO
	OIDCManager *utils.OIDCManager
}

func NewOAuthService(userRepo *repository.UserRepository, oauthRepo *repository.OAuthRepository, imgMinio *storage.MinIO, oidc *utils.OIDCManager) *OAuthService {
	return &OAuthService{
		UserRepo:    userRepo,
		OAuthRepo:   oauthRepo,
		ImgMinio:    imgMinio,
		OIDCManager: oidc,
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
	if err != nil {
		return nil, errors.New("密钥加密失败")
	}
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
	res, err := o.OAuthRepo.GetOAuthClientInfoByClientID(ctx, clientID)
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
func (o *OAuthService) ResetOAuthClientSecret(ctx context.Context, clientID string, userID uint64) (string, error) {
	err := o.CheckLevel(ctx, userID)
	if err != nil {
		return "", err
	}
	_, err = o.OAuthRepo.GetOAuthClientByClientID(ctx, clientID)
	if err != nil {
		return "", errors.New("应用不存在")
	}
	secret, maskSecret, err := utils.GenerateClientSecret()
	if err != nil {
		return "", err
	}

	hashSecret, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	if err != nil {
		return "", errors.New("密钥加密失败")
	}
	err = o.OAuthRepo.ResetOAuthClientSecret(ctx, clientID, string(hashSecret), maskSecret)
	if err != nil {
		return "", err
	}
	return secret, nil
}

func (o *OAuthService) Authorize(ctx context.Context, userID uint64, req request.AuthorizeReq) (string, error) {
	client, err := o.OAuthRepo.GetOAuthClientByClientID(ctx, req.ClientID)
	if err != nil {
		return "", errors.New("应用不存在")
	}
	if !client.IsActive {
		return "", errors.New("应用已停用")
	}
	allowedURIs := strings.Split(client.RedirectURIs, ",")
	isMatch := false
	for _, u := range allowedURIs {
		if strings.TrimSpace(u) == req.RedirectURI {
			isMatch = true
			break
		}
	}
	if !isMatch {
		return "", errors.New("redirect_uri 不合法")
	}
	if req.Scope == "" {
		req.Scope = client.Scope
	}
	code := "oaa_code_" + strings.ReplaceAll(uuid.NewV7().String(), "-", "")
	codeData := model.AuthCodeData{
		UserID:      userID,
		ClientID:    req.ClientID,
		RedirectURI: req.RedirectURI,
		Scope:       req.Scope,
	}
	err = o.OAuthRepo.SaveAuthCode(ctx, code, codeData, 5*time.Minute)
	if err != nil {
		return "", errors.New("生成授权码失败")
	}
	parsedURL, err := url.Parse(req.RedirectURI)
	if err != nil {
		return "", errors.New("解析回调地址失败")
	}
	query := parsedURL.Query()
	query.Set("code", code)
	if req.State != "" {
		query.Set("state", req.State)
	}
	parsedURL.RawQuery = query.Encode()
	return parsedURL.String(), nil
}

func (o *OAuthService) ExchangeCode(ctx context.Context, req request.TokenReq) (*request.TokenRes, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	client, err := o.OAuthRepo.GetOAuthClientByClientID(ctx, req.ClientID)
	if err != nil {
		return nil, errors.New("应用不存在")
	}
	if !client.IsActive {
		return nil, errors.New("应用已停用")
	}
	err = bcrypt.CompareHashAndPassword([]byte(client.ClientSecret), []byte(req.ClientSecret))
	if err != nil {
		return nil, errors.New("客户端密钥错误")
	}
	key := "oauth:code:" + req.Code
	code, err := o.OAuthRepo.GetAuthCodeByKey(ctx, key)
	if err != nil {
		return nil, errors.New("获取 auth code失败")
	}
	var codeData model.AuthCodeData
	if err := json.Unmarshal([]byte(code), &codeData); err != nil {
		return nil, errors.New("解析授权码失败")
	}
	if codeData.ClientID != req.ClientID {
		return nil, errors.New("授权码和客户端不匹配")
	}
	if codeData.RedirectURI != req.RedirectURI {
		return nil, errors.New("redirectURI不匹配")
	}
	accessToken := "oaa_at_" + strings.ReplaceAll(uuid.NewV7().String(), "-", "")
	tokenData, _ := json.Marshal(map[string]any{
		"user_id":   codeData.UserID,
		"client_id": codeData.ClientID,
		"scope":     codeData.Scope,
	})
	err = o.OAuthRepo.SetToken(ctx, accessToken, tokenData, 2*time.Hour)
	if err != nil {
		return nil, errors.New("保存 Access Token 失败")
	}
	hasScope := func(rawScope, target string) bool {
		return slices.Contains(strings.Fields(rawScope), target)
	}

	var idToken string
	if hasScope(codeData.Scope, "openid") {
		extraClaims := map[string]any{}
		userInfo, err := o.UserRepo.GetUserInfoById(ctx, codeData.UserID)
		if err == nil {
			if hasScope(codeData.Scope, "profile") {
				extraClaims["name"] = userInfo.Name
				extraClaims["preferred_username"] = userInfo.Name
				extraClaims["student_id"] = userInfo.StudentID
			}
			if hasScope(codeData.Scope, "email") {
				extraClaims["email"] = userInfo.Email
			}
		}

		idToken, err = o.OIDCManager.GenerateIDToken(
			strconv.FormatUint(codeData.UserID, 10),
			req.ClientID,
			2*time.Hour,
			extraClaims,
		)
		if err != nil {
			return nil, errors.New("生成 id_token 失败")
		}
	}
	return &request.TokenRes{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   7200,
		Scope:       codeData.Scope,
		IDToken:     idToken,
	}, nil
}

func (o *OAuthService) GetUserInfo(ctx context.Context, token string) (*request.UserInfoRes, error) {
	tokenStr, err := o.OAuthRepo.GetTokenByKey(ctx, token)
	if err != nil {
		return nil, errors.New("access token 无效或者过期")
	}
	var tokenData struct {
		UserID   uint64 `json:"user_id"`
		ClientID string `json:"client_id"`
		Scope    string `json:"scope"`
	}
	if err := json.Unmarshal([]byte(tokenStr), &tokenData); err != nil {
		return nil, errors.New("access token数据损坏")
	}
	userInfo, err := o.UserRepo.GetUserInfoById(ctx, tokenData.UserID)
	if err != nil {
		return nil, errors.New("用户不存在")
	}
	hasScope := func(rawScope, target string) bool {
		return slices.Contains(strings.Fields(rawScope), target)
	}
	resp := &request.UserInfoRes{
		Sub: strconv.FormatUint(tokenData.UserID, 10),
	}
	if hasScope(tokenData.Scope, "profile") {
		resp.Name = userInfo.Name
		resp.PreferredUsername = userInfo.Name
		resp.StudentID = userInfo.StudentID
	}
	if hasScope(tokenData.Scope, "email") {
		resp.Email = userInfo.Email
	}
	if hasScope(tokenData.Scope, "level") {
		resp.Department = userInfo.Department
		resp.Role = userInfo.Role
		resp.Level = userInfo.RoleLevel
	}
	return resp, nil
}

func (o *OAuthService) GetPublicClientInfo(ctx context.Context, clientID string, requestedScope string) (*request.PublicClientInfoRes, error) {
	client, err := o.OAuthRepo.GetOAuthClientByClientID(ctx, clientID)
	if err != nil {
		return nil, errors.New("应用不存在")
	}
	if !client.IsActive {
		return nil, errors.New("应用已停用")
	}

	targetScope := requestedScope
	if targetScope == "" {
		targetScope = client.Scope
	}

	scopeKeys := strings.Fields(targetScope)
	var scopes []request.ScopeDetail
	for _, key := range scopeKeys {
		if detail, ok := request.ScopeDictionary[key]; ok {
			scopes = append(scopes, detail)
		}
	}

	return &request.PublicClientInfoRes{
		ClientID:   client.ClientID,
		ClientName: client.Name,
		Scopes:     scopes,
	}, nil
}
