package service

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"suseoaa/internal/storage"
	"suseoaa/pkg/utils"
)

type FileService struct {
	Storage *storage.MinIO
}

func NewFileService(storage *storage.MinIO) FileService {
	return FileService{
		Storage: storage,
	}
}

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
	err = f.Storage.UploadFile(ctx, objectName, fileValue, file.Size, file.Header.Get("Content-Type"))
	if err != nil {
		return nil, err
	}
	presignedURL, err := f.Storage.GeneratePresignedURL(ctx, objectName)
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
	err = f.Storage.UploadFile(ctx, objectName, fileValue, file.Size, file.Header.Get("Content-Type"))
	if err != nil {
		return nil, err
	}
	presignedURL, err := f.Storage.GeneratePresignedURL(ctx, objectName)
	if err != nil {
		return nil, err
	}
	return map[string]string{
		"uri": objectName,
		"url": presignedURL,
	}, nil
}
