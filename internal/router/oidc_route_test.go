package router

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"suseoaa/internal/handler"
	"suseoaa/internal/service"
	"suseoaa/pkg/utils"

	"github.com/gin-gonic/gin"
)

func TestRouter_OpenIDConfigurationAndJWKS(t *testing.T) {
	gin.SetMode(gin.TestMode)

	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate rsa key: %v", err)
	}
	oidcManager := &utils.OIDCManager{
		PrivateKey: privKey,
		PublicKey:  &privKey.PublicKey,
		KeyID:      "test-key-id",
		Issuer:     "https://v2-api.suseoaa.com",
	}

	oauthService := &service.OAuthService{
		OIDCManager: oidcManager,
	}
	oauthHandler := handler.NewOAuthHandler(oauthService, "test_secret")

	total := &handler.TotalHandler{
		Health: handler.NewHealthHandler(nil, nil),
		Auth:   &handler.AuthHandler{JwtSecret: "test_secret"},
		OAuth:  oauthHandler,
	}

	r := RouterInit(total, nil, nil)

	// Test 1: GET /.well-known/openid-configuration
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest("GET", "/.well-known/openid-configuration", nil)
	r.ServeHTTP(w1, req1)
	if w1.Code != 200 {
		t.Fatalf("expected 200 for openid-configuration, got %d", w1.Code)
	}

	var conf utils.OpenIDConfiguration
	if err := json.Unmarshal(w1.Body.Bytes(), &conf); err != nil {
		t.Fatalf("failed to parse openid configuration json: %v", err)
	}
	if conf.Issuer != "https://v2-api.suseoaa.com" {
		t.Errorf("expected issuer https://v2-api.suseoaa.com, got %s", conf.Issuer)
	}
	if conf.AuthorizationEndpoint != "https://v2-api.suseoaa.com/v2/oauth/authorize" {
		t.Errorf("unexpected authorization_endpoint: %s", conf.AuthorizationEndpoint)
	}
	if conf.TokenEndpoint != "https://v2-api.suseoaa.com/v2/oauth/token" {
		t.Errorf("unexpected token_endpoint: %s", conf.TokenEndpoint)
	}
	if conf.UserinfoEndpoint != "https://v2-api.suseoaa.com/v2/oauth/userinfo" {
		t.Errorf("unexpected userinfo_endpoint: %s", conf.UserinfoEndpoint)
	}
	if conf.JwksURI != "https://v2-api.suseoaa.com/.well-known/jwks.json" {
		t.Errorf("unexpected jwks_uri: %s", conf.JwksURI)
	}

	// Test 2: GET /.well-known/jwks.json
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/.well-known/jwks.json", nil)
	r.ServeHTTP(w2, req2)
	if w2.Code != 200 {
		t.Fatalf("expected 200 for jwks.json, got %d", w2.Code)
	}

	var jwks utils.JWKS
	if err := json.Unmarshal(w2.Body.Bytes(), &jwks); err != nil {
		t.Fatalf("failed to parse jwks json: %v", err)
	}
	if len(jwks.Keys) != 1 || jwks.Keys[0].Kid != "test-key-id" {
		t.Errorf("unexpected jwks keys: %+v", jwks.Keys)
	}

	// Test 3: POST /v2/oauth/token (Route mounted under v2)
	w3 := httptest.NewRecorder()
	req3 := httptest.NewRequest("POST", "/v2/oauth/token", nil)
	r.ServeHTTP(w3, req3)
	if w3.Code == 404 {
		t.Fatalf("expected /v2/oauth/token to be mounted, got 404")
	}
}
