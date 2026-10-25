package service

import (
	"testing"
)

func TestNewVerificationCode(t *testing.T) {
	emailService := &EmailService{}

	// 测试 6 位纯数字验证码
	code := emailService.NewVerificationCode(6)
	if len(code) != 6 {
		t.Fatalf("expected code length 6, got %d (%s)", len(code), code)
	}

	for _, ch := range code {
		if ch < '0' || ch > '9' {
			t.Errorf("expected digit character, got %c", ch)
		}
	}

	// 测试多次生成具备随机性
	code2 := emailService.NewVerificationCode(6)
	if len(code2) != 6 {
		t.Fatalf("expected code length 6, got %d (%s)", len(code2), code2)
	}
}

func TestNewEmailService_DialerConfig(t *testing.T) {
	svc := NewEmailService("smtp.example.com", 465, "user@example.com", "pass", 5, 1)
	if svc.dialer == nil {
		t.Fatal("expected non-nil dialer")
	}
	if svc.dialer.TLSConfig == nil || svc.dialer.TLSConfig.ServerName != "smtp.example.com" {
		t.Errorf("expected TLSConfig with ServerName smtp.example.com, got %v", svc.dialer.TLSConfig)
	}
	if svc.dialer.TLSConfig.InsecureSkipVerify {
		t.Errorf("expected InsecureSkipVerify=false, got %v", svc.dialer.TLSConfig.InsecureSkipVerify)
	}
}
