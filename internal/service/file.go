package service

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"regexp"
	"strings"
	"suseoaa/internal/storage"
	"suseoaa/pkg/utils"
)

type FileService struct {
	ImgStorage  *storage.MinIO
	FileStorage *storage.MinIO
}

func NewFileService(imgStorage *storage.MinIO, fileStorage *storage.MinIO) FileService {
	return FileService{
		ImgStorage:  imgStorage,
		FileStorage: fileStorage,
	}
}

var ossRegexp = regexp.MustCompile(`oss://[^\s)\]>\"']+`)
var allowedExt = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".webp": true,
	".gif":  true,
	".avif": true,
}

func (f *FileService) UploadImage(ctx context.Context, file *multipart.FileHeader, scene string) (map[string]string, error) {
	if file.Size > storage.MaxImageSize {
		return nil, errors.New("图片体积过大")
	}
	ext := filepath.Ext(file.Filename)
	if !allowedExt[ext] {
		return nil, errors.New("图片类型错误")
	}
	uuid, err := utils.GetUUID()
	if err != nil {
		return nil, errors.New("创建 uuid 失败" + err.Error())
	}
	fileValue, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer fileValue.Close()
	objectName := fmt.Sprintf("%s/%s%s", scene, uuid, ext)
	err = f.ImgStorage.UploadFile(ctx, objectName, fileValue, file.Size, file.Header.Get("Content-Type"))
	if err != nil {
		return nil, err
	}
	presignedURL, err := f.ImgStorage.GeneratePresignedURL(ctx, objectName)
	if err != nil {
		return nil, err
	}
	return map[string]string{
		"uri": objectName,
		"url": presignedURL,
	}, nil
}
func (f *FileService) UploadFile(ctx context.Context, file *multipart.FileHeader, scene string) (map[string]string, error) {
	if file.Size > storage.MaxFileSize {
		return nil, errors.New("文件体积过大")
	}
	uuid, err := utils.GetUUID()
	if err != nil {
		return nil, errors.New("创建 uuid 失败" + err.Error())
	}
	fileValue, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer fileValue.Close()
	ext := filepath.Ext(file.Filename)
	objectName := fmt.Sprintf("%s/%s%s", scene, uuid, ext)
	err = f.FileStorage.UploadFile(ctx, objectName, fileValue, file.Size, file.Header.Get("Content-Type"))
	if err != nil {
		return nil, err
	}
	presignedURL, err := f.FileStorage.GeneratePresignedURL(ctx, objectName)
	if err != nil {
		return nil, err
	}
	return map[string]string{
		"uri": objectName,
		"url": presignedURL,
	}, nil
}
func (f *FileService) ReplaceMinIOLinks(ctx context.Context, content string) (string, error) {
	urlCache := make(map[string]string)
	var firstErr error
	replacedContent := ossRegexp.ReplaceAllStringFunc(content, func(matched string) string {
		if firstErr != nil {
			return matched
		}
		if err := ctx.Err(); err != nil {
			firstErr = err
			return matched
		}
		if presignedURL, exists := urlCache[matched]; exists {
			return presignedURL
		}
		rawPath := strings.TrimPrefix(matched, "oss://")

		parts := strings.SplitN(rawPath, "/", 2)
		if len(parts) < 2 || parts[1] == "" {
			return matched
		}
		bucketName, objectName := parts[0], parts[1]

		var targetStorage *storage.MinIO
		switch {
		case f.ImgStorage != nil && bucketName == f.ImgStorage.Bucket:
			targetStorage = f.ImgStorage
		case f.FileStorage != nil && bucketName == f.FileStorage.Bucket:
			targetStorage = f.FileStorage
		default:
			return matched
		}

		presignedURL, err := targetStorage.GeneratePresignedURL(ctx, objectName)
		if err != nil {
			firstErr = fmt.Errorf("failed to sign url for [%s]: %w", rawPath, err)
			return matched
		}

		urlCache[matched] = presignedURL
		return presignedURL
	})

	if firstErr != nil {
		return "", firstErr
	}
	return replacedContent, nil
}
