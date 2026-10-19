package config

import (
	"testing"
)

func TestConfigInit_Normal(t *testing.T) {
	cfg := ConfigInit()

	if cfg.Server.Port == "" {
		t.Fatalf("expected server.port to be non-empty")
	}
	if cfg.Mysql.Database == "" {
		t.Fatalf("expected mysql.database to be non-empty")
	}
}

func TestConfigInit_EnvOverride(t *testing.T) {

	const mockPassword = "env_injected_super_secure_password_999"
	const mockPort = "18080"
	const mockSecret = "env_jwt_secret_injected_777"

	t.Setenv("OAA_MYSQL_PASSWORD", mockPassword)
	t.Setenv("OAA_SERVER_PORT", mockPort)
	t.Setenv("OAA_JWT_SECRET", mockSecret)

	cfg := ConfigInit()

	if cfg.Mysql.Password != mockPassword {
		t.Fatalf("expected mysql.password to be overridden by env, want %q, got %q", mockPassword, cfg.Mysql.Password)
	}
	if cfg.Server.Port != mockPort {
		t.Fatalf("expected server.port to be overridden by env, want %q, got %q", mockPort, cfg.Server.Port)
	}
	if cfg.Jwt.Secret != mockSecret {
		t.Fatalf("expected jwt.secret to be overridden by env, want %q, got %q", mockSecret, cfg.Jwt.Secret)
	}
}
