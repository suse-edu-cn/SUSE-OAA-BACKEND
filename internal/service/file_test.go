package service

import (
	"context"
	"errors"
	"mime/multipart"
	"strings"
	"suseoaa/internal/storage"
	"testing"
)

func TestIsValidScene(t *testing.T) {
	tests := []struct {
		name     string
		scene    string
		expected bool
	}{
		{"正常纯字母", "avatar", true},
		{"包含下划线", "user_avatar", true},
		{"包含中划线", "announcement-banner", true},
		{"字母数字混合", "scene123", true},
		{"大写字母", "AvatarImage", true},
		{"下划线与中划线组合", "my_custom-scene_01", true},
		{"空字符串", "", false},
		{"包含点号(相对路径前缀)", "./avatar", false},
		{"包含路径穿越..", "../avatar", false},
		{"仅点号.", ".", false},
		{"双点..", "..", false},
		{"包含斜杠/", "avatar/sub", false},
		{"包含反斜杠\\", "avatar\\sub", false},
		{"包含空格", "avatar scene", false},
		{"包含特殊符号$", "avatar$", false},
		{"包含特殊符号@", "user@scene", false},
		{"包含中文", "头像", false},
		{"超过64字符限制", strings.Repeat("a", 65), false},
		{"正好64字符", strings.Repeat("a", 64), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidScene(tt.scene)
			if got != tt.expected {
				t.Errorf("IsValidScene(%q) = %v; expected %v", tt.scene, got, tt.expected)
			}
		})
	}
}

func TestUploadImage_Validation(t *testing.T) {
	storage.MaxImageSize = 5 * 1024 * 1024 // 5MB
	svc := &FileService{}
	ctx := context.Background()

	// 1. 测试非法 scene（包含 .、/ 等危险字符）
	invalidScenes := []string{"../attack", "avatar/test", "avatar.dot", "", "avatar\\win"}
	for _, scene := range invalidScenes {
		_, err := svc.UploadImage(ctx, &multipart.FileHeader{Filename: "test.png", Size: 100}, scene)
		if !errors.Is(err, ErrInvalidScene) {
			t.Errorf("scene=%q: expected ErrInvalidScene, got %v", scene, err)
		}
	}

	// 2. 测试图片体积过大
	largeHeader := &multipart.FileHeader{
		Filename: "big.png",
		Size:     storage.MaxImageSize + 1,
	}
	_, err := svc.UploadImage(ctx, largeHeader, "avatar")
	if !errors.Is(err, ErrImageTooLarge) {
		t.Errorf("expected ErrImageTooLarge, got %v", err)
	}

	// 3. 测试不支持的图片后缀
	invalidExtHeader := &multipart.FileHeader{
		Filename: "evil.exe",
		Size:     1024,
	}
	_, err = svc.UploadImage(ctx, invalidExtHeader, "avatar")
	if !errors.Is(err, ErrInvalidImageType) {
		t.Errorf("expected ErrInvalidImageType, got %v", err)
	}
}

func TestUploadFile_Validation(t *testing.T) {
	storage.MaxFileSize = 50 * 1024 * 1024 // 50MB
	svc := &FileService{}
	ctx := context.Background()

	// 1. 测试非法 scene
	invalidScenes := []string{"../attack", "file/sub", "file.doc", "", "bad\\path"}
	for _, scene := range invalidScenes {
		_, err := svc.UploadFile(ctx, &multipart.FileHeader{Filename: "test.pdf", Size: 100}, scene)
		if !errors.Is(err, ErrInvalidScene) {
			t.Errorf("scene=%q: expected ErrInvalidScene, got %v", scene, err)
		}
	}

	// 2. 测试文件体积过大
	largeHeader := &multipart.FileHeader{
		Filename: "big.zip",
		Size:     storage.MaxFileSize + 1,
	}
	_, err := svc.UploadFile(ctx, largeHeader, "material")
	if !errors.Is(err, ErrFileTooLarge) {
		t.Errorf("expected ErrFileTooLarge, got %v", err)
	}
}
