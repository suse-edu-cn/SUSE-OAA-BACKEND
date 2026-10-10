package utils

import (
	"testing"
)

func TestGenerateAndParseToken(t *testing.T) {
	secret := "test-secret-key-12345"
	name := "张三"
	userID := uint64(1001)
	studentID := "20240101"
	expireMinute := 60

	tokenStr, err := GenerateToken(name, userID, studentID, secret, expireMinute)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}
	if tokenStr == "" {
		t.Fatal("token string is empty")
	}

	claims, err := ParseToken(tokenStr, secret)
	if err != nil {
		t.Fatalf("ParseToken failed: %v", err)
	}

	if claims.Name != name {
		t.Errorf("expected name %s, got %s", name, claims.Name)
	}
	if claims.UserID != userID {
		t.Errorf("expected userID %d, got %d", userID, claims.UserID)
	}
	if claims.StudentID != studentID {
		t.Errorf("expected studentID %s, got %s", studentID, claims.StudentID)
	}
	if claims.Issuer != "GrowthOS" {
		t.Errorf("expected issuer GrowthOS, got %s", claims.Issuer)
	}
}

func TestParseToken_Expired(t *testing.T) {
	secret := "test-secret-key-12345"
	// 传负数过期时间生成已过期 Token
	tokenStr, err := GenerateToken("李四", 1002, "20240102", secret, -1)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	_, err = ParseToken(tokenStr, secret)
	if err == nil {
		t.Fatal("expected error for expired token, but got nil")
	}
}

func TestParseToken_InvalidSecret(t *testing.T) {
	secret := "test-secret-key-12345"
	wrongSecret := "wrong-secret-key-99999"
	tokenStr, err := GenerateToken("王五", 1003, "20240103", secret, 30)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	_, err = ParseToken(tokenStr, wrongSecret)
	if err == nil {
		t.Fatal("expected error for invalid secret, but got nil")
	}
}

func TestParseToken_Malformed(t *testing.T) {
	_, err := ParseToken("not-a-valid-jwt-token", "secret")
	if err == nil {
		t.Fatal("expected error for malformed token, but got nil")
	}
}
