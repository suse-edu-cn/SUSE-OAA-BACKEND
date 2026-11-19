package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"suseoaa/internal/model"
	"suseoaa/internal/repository"
	"suseoaa/internal/request"
	"suseoaa/internal/storage"
	"suseoaa/pkg/utils"

	"github.com/alicebob/miniredis/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestOAuthService(t *testing.T, withMinIO bool) (*OAuthService, *gorm.DB, *repository.OAuthRepository) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite in-memory db: %v", err)
	}

	if err := db.AutoMigrate(&model.User{}, &model.Department{}, &model.Role{}, &model.Interviewer{}, &model.OAuthClient{}); err != nil {
		t.Fatalf("failed to migrate tables: %v", err)
	}

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	t.Cleanup(func() {
		_ = rdb.Close()
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	})

	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate rsa key: %v", err)
	}

	oidcManager := &utils.OIDCManager{
		PrivateKey: privKey,
		PublicKey:  &privKey.PublicKey,
		KeyID:      "test-key",
		Issuer:     "https://v2-api.suseoaa.com",
	}

	var imgMinio *storage.MinIO
	if withMinIO {
		imgMinio, _ = storage.NewMinIO(
			"127.0.0.1:9000",
			"https://obj.suseoaa.com",
			"us-east-1",
			"test-access-key",
			"test-secret-key",
			false,
			true,
			"oaa-img",
			"oaa-file",
			10,
			5,
			15,
		)
	}

	userRepo := repository.NewUserRepository(db, rdb)
	oauthRepo := repository.NewOAuthRepository(db, rdb)
	svc := NewOAuthService(userRepo, oauthRepo, imgMinio, oidcManager)
	return svc, db, oauthRepo
}

func seedOAuthTestUser(t *testing.T, db *gorm.DB, avatar string) model.User {
	t.Helper()

	dept := model.Department{Name: "技术部", Type: "技术", IsActive: true}
	if err := db.Create(&dept).Error; err != nil {
		t.Fatalf("failed to create department: %v", err)
	}
	role := model.Role{Name: "部长", Type: "技术", Level: 60, IsActive: true}
	if err := db.Create(&role).Error; err != nil {
		t.Fatalf("failed to create role: %v", err)
	}

	user := model.User{
		StudentID:    "20240101001",
		Username:     "starry_nick",
		Name:         "张三",
		Email:        "zhangsan@suseoaa.com",
		Password:     "hashed",
		DepartmentID: dept.ID,
		RoleID:       role.ID,
		Avatar:       avatar,
		Grade:        2024,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	return user
}

func TestOAuthService_GetUserInfo_ProfileClaims(t *testing.T) {
	svc, db, oauthRepo := setupTestOAuthService(t, true)
	ctx := context.Background()
	user := seedOAuthTestUser(t, db, "avatar/custom-avatar.png")

	tokenData, _ := json.Marshal(map[string]any{
		"user_id":   user.ID,
		"client_id": "oj_client",
		"scope":     "openid profile",
	})
	if err := oauthRepo.SetToken(ctx, "oaa_at_profile", tokenData, time.Hour); err != nil {
		t.Fatalf("failed to set token: %v", err)
	}

	res, err := svc.GetUserInfo(ctx, "oaa_at_profile")
	if err != nil {
		t.Fatalf("GetUserInfo failed: %v", err)
	}

	if res.Sub != strconv.FormatUint(user.ID, 10) {
		t.Errorf("unexpected sub: got %q, want %d", res.Sub, user.ID)
	}
	if res.Name != "张三" {
		t.Errorf("unexpected name: got %q, want %q", res.Name, "张三")
	}
	if res.PreferredUsername != "starry_nick" {
		t.Errorf("unexpected preferred_username: got %q, want %q", res.PreferredUsername, "starry_nick")
	}
	if res.Nickname != "starry_nick" {
		t.Errorf("unexpected nickname: got %q, want %q", res.Nickname, "starry_nick")
	}
	if res.StudentID != "20240101001" {
		t.Errorf("unexpected student_id: got %q, want %q", res.StudentID, "20240101001")
	}

	parsedPic, err := url.Parse(res.Picture)
	if err != nil || parsedPic.Scheme != "https" || parsedPic.Host != "obj.suseoaa.com" {
		t.Fatalf("expected valid presigned picture URL, got %q (err=%v)", res.Picture, err)
	}
	if !strings.Contains(parsedPic.Path, "/oaa-img/avatar/custom-avatar.png") {
		t.Errorf("unexpected picture path: %q", parsedPic.Path)
	}

	if res.Email != "" || res.Department != "" || res.Role != "" || res.Level != 0 {
		t.Errorf("expected non-profile claims to be omitted, got %+v", res)
	}
}

func TestOAuthService_GetUserInfo_DefaultAvatarAndNilStorage(t *testing.T) {
	t.Run("空头像兜底默认头像预签名链接", func(t *testing.T) {
		svc, db, oauthRepo := setupTestOAuthService(t, true)
		ctx := context.Background()
		user := seedOAuthTestUser(t, db, "   ")

		tokenData, _ := json.Marshal(map[string]any{
			"user_id":   user.ID,
			"client_id": "oj_client",
			"scope":     "openid profile",
		})
		_ = oauthRepo.SetToken(ctx, "oaa_at_default_avatar", tokenData, time.Hour)

		res, err := svc.GetUserInfo(ctx, "oaa_at_default_avatar")
		if err != nil {
			t.Fatalf("GetUserInfo failed: %v", err)
		}
		if !strings.Contains(res.Picture, "/oaa-img/avatar/default.png") {
			t.Errorf("expected fallback default avatar URL, got %q", res.Picture)
		}
	})

	t.Run("未初始化 ImgMinio 时省略 picture 而非返回相对路径", func(t *testing.T) {
		svc, db, oauthRepo := setupTestOAuthService(t, false)
		ctx := context.Background()
		user := seedOAuthTestUser(t, db, "avatar/custom.png")

		tokenData, _ := json.Marshal(map[string]any{
			"user_id":   user.ID,
			"client_id": "oj_client",
			"scope":     "openid profile",
		})
		_ = oauthRepo.SetToken(ctx, "oaa_at_no_minio", tokenData, time.Hour)

		res, err := svc.GetUserInfo(ctx, "oaa_at_no_minio")
		if err != nil {
			t.Fatalf("GetUserInfo failed: %v", err)
		}
		if res.Picture != "" {
			t.Errorf("expected empty picture when ImgMinio is nil, got %q", res.Picture)
		}
	})

	t.Run("未授权 profile 时不返回 profile 字段", func(t *testing.T) {
		svc, db, oauthRepo := setupTestOAuthService(t, true)
		ctx := context.Background()
		user := seedOAuthTestUser(t, db, "avatar/custom.png")

		tokenData, _ := json.Marshal(map[string]any{
			"user_id":   user.ID,
			"client_id": "oj_client",
			"scope":     "openid email level",
		})
		_ = oauthRepo.SetToken(ctx, "oaa_at_no_profile", tokenData, time.Hour)

		res, err := svc.GetUserInfo(ctx, "oaa_at_no_profile")
		if err != nil {
			t.Fatalf("GetUserInfo failed: %v", err)
		}
		if res.Name != "" || res.PreferredUsername != "" || res.Nickname != "" || res.Picture != "" || res.StudentID != "" {
			t.Errorf("expected profile claims to be empty when scope excludes profile, got %+v", res)
		}
		if res.Email != "zhangsan@suseoaa.com" || res.Department != "技术部" || res.Role != "部长" || res.Level != 60 {
			t.Errorf("expected email and level claims to be populated, got %+v", res)
		}
	})
}

func TestOAuthService_ExchangeCode_IDTokenProfileClaims(t *testing.T) {
	svc, db, oauthRepo := setupTestOAuthService(t, true)
	ctx := context.Background()
	user := seedOAuthTestUser(t, db, "avatar/custom.png")

	rawSecret := "super-secret"
	hashSecret, err := bcrypt.GenerateFromPassword([]byte(rawSecret), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("failed to hash client secret: %v", err)
	}
	client := model.OAuthClient{
		ID:           "client-uuid-1",
		ClientID:     "hydro_oj",
		ClientSecret: string(hashSecret),
		MaskSecret:   "supe****cret",
		Name:         "OJ 平台",
		RedirectURIs: "https://oj.suseoaa.com/callback",
		Scope:        "openid profile email",
		IsActive:     true,
	}
	if err := db.Create(&client).Error; err != nil {
		t.Fatalf("failed to create oauth client: %v", err)
	}

	code := "oaa_code_test123"
	err = oauthRepo.SaveAuthCode(ctx, code, model.AuthCodeData{
		UserID:      user.ID,
		ClientID:    client.ClientID,
		RedirectURI: "https://oj.suseoaa.com/callback",
		Scope:       "openid profile",
	}, 5*time.Minute)
	if err != nil {
		t.Fatalf("failed to save auth code: %v", err)
	}

	res, err := svc.ExchangeCode(ctx, request.TokenReq{
		GrantType:    "authorization_code",
		ClientID:     client.ClientID,
		ClientSecret: rawSecret,
		Code:         code,
		RedirectURI:  "https://oj.suseoaa.com/callback",
	})
	if err != nil {
		t.Fatalf("ExchangeCode failed: %v", err)
	}
	if res.IDToken == "" {
		t.Fatal("expected non-empty id_token")
	}

	parsedToken, err := jwt.Parse(res.IDToken, func(token *jwt.Token) (any, error) {
		return svc.OIDCManager.PublicKey, nil
	})
	if err != nil || !parsedToken.Valid {
		t.Fatalf("failed to verify id_token: %v", err)
	}
	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("failed to cast claims to jwt.MapClaims")
	}

	if claims["name"] != "张三" {
		t.Errorf("unexpected id_token name: %v", claims["name"])
	}
	if claims["preferred_username"] != "starry_nick" {
		t.Errorf("unexpected id_token preferred_username: %v", claims["preferred_username"])
	}
	if claims["nickname"] != "starry_nick" {
		t.Errorf("unexpected id_token nickname: %v", claims["nickname"])
	}
	if claims["student_id"] != "20240101001" {
		t.Errorf("unexpected id_token student_id: %v", claims["student_id"])
	}
	if _, exists := claims["picture"]; exists {
		t.Errorf("did not expect presigned picture URL embedded in id_token")
	}
	if !slices.Contains(svc.OIDCManager.GetOpenIDConfiguration("").ClaimsSupported, "nickname") ||
		!slices.Contains(svc.OIDCManager.GetOpenIDConfiguration("").ClaimsSupported, "picture") {
		t.Errorf("expected nickname and picture in claims_supported")
	}
}
