package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var MaxFileSize int64
var MaxImageSize int64
var ImgBucketName string
var FileBucketName string
var expire time.Duration

type MinIO struct {
	Client *minio.Client
	Bucket string
}

func NewMinIO(
	endpoint string,
	accessKey string,
	secretKey string,
	useSSL bool,
	imgBucket string,
	fileBucket string,
	maxFileSize int64,
	maxImageSize int64,
	expireTime int64,
) (*MinIO, *MinIO) {
	imgClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	fileClient, err := minio.New(endpoint, &minio.Options{
		Creds: credentials.NewStaticV4(accessKey, secretKey, ""),
	})
	if err != nil {
		panic(err)
	}
	MaxFileSize = maxFileSize * 1024 * 1024
	MaxImageSize = maxImageSize * 1024 * 1024
	ImgBucketName = imgBucket
	FileBucketName = fileBucket
	expire = time.Duration(expireTime) * time.Minute
	return &MinIO{
			Client: imgClient,
			Bucket: imgBucket,
		}, &MinIO{
			Client: fileClient,
			Bucket: fileBucket,
		}
}

func (m *MinIO) UploadFile(
	ctx context.Context,
	objectName string,
	reader io.Reader,
	size int64,
	contentType string) error {
	_, err := m.Client.PutObject(ctx, m.Bucket, objectName, reader, size, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return fmt.Errorf("upload file: %w", err)
	}
	return nil
}
func (m *MinIO) GeneratePresignedURL(
	ctx context.Context,
	objectName string,
) (string, error) {
	url, err := m.Client.PresignedGetObject(
		ctx,
		m.Bucket,
		objectName,
		expire,
		nil,
	)
	if err != nil {
		return "", err
	}
	return url.String(), nil
}
func (m *MinIO) GetFileInfo(ctx context.Context, objectName string) (int64, error) {
	info, err := m.Client.StatObject(
		ctx,
		m.Bucket,
		objectName,
		minio.StatObjectOptions{},
	)
	if err != nil {
		return 0, fmt.Errorf("get file info: %w", err)
	}

	return info.Size, nil
}
func (m *MinIO) DeleteFile(ctx context.Context, objectName string) error {
	err := m.Client.RemoveObject(
		ctx,
		m.Bucket,
		objectName,
		minio.RemoveObjectOptions{},
	)
	if err != nil {
		return fmt.Errorf("delete file: %w", err)
	}
	return nil
}
