package storage

import (
	"context"
	"fmt"
	"io"
	"strings"
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
	Client     *minio.Client
	SignClient *minio.Client
	Bucket     string
}

func cleanEndpoint(rawEndpoint string, useSSL bool) (string, bool) {
	rawEndpoint = strings.TrimSpace(rawEndpoint)
	if strings.HasPrefix(rawEndpoint, "https://") {
		return strings.TrimRight(strings.TrimPrefix(rawEndpoint, "https://"), "/"), true
	}
	if strings.HasPrefix(rawEndpoint, "http://") {
		return strings.TrimRight(strings.TrimPrefix(rawEndpoint, "http://"), "/"), false
	}
	return strings.TrimRight(rawEndpoint, "/"), useSSL
}

func NewMinIO(
	endpoint string,
	publicEndpoint string,
	accessKey string,
	secretKey string,
	useSSL bool,
	publicUseSSL bool,
	imgBucket string,
	fileBucket string,
	maxFileSize int64,
	maxImageSize int64,
	expireTime int64,
) (*MinIO, *MinIO) {
	endpoint, useSSL = cleanEndpoint(endpoint, useSSL)

	imgClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
		Region: "us-east-1",
	})
	if err != nil {
		panic(fmt.Errorf("init minio img client failed: %w", err))
	}
	fileClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
		Region: "us-east-1",
	})
	if err != nil {
		panic(fmt.Errorf("init minio file client failed: %w", err))
	}

	signEndpoint := endpoint
	signUseSSL := useSSL
	if publicEndpoint != "" {
		signEndpoint, signUseSSL = cleanEndpoint(publicEndpoint, publicUseSSL)
	}

	imgSignClient := imgClient
	fileSignClient := fileClient
	if signEndpoint != endpoint || signUseSSL != useSSL {
		imgSignClient, err = minio.New(signEndpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
			Secure: signUseSSL,
			Region: "us-east-1",
		})
		if err != nil {
			panic(fmt.Errorf("init minio img sign client failed: %w", err))
		}
		fileSignClient, err = minio.New(signEndpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
			Secure: signUseSSL,
			Region: "us-east-1",
		})
		if err != nil {
			panic(fmt.Errorf("init minio file sign client failed: %w", err))
		}
	}

	MaxFileSize = maxFileSize * 1024 * 1024
	MaxImageSize = maxImageSize * 1024 * 1024
	ImgBucketName = imgBucket
	FileBucketName = fileBucket
	expire = time.Duration(expireTime) * time.Minute
	return &MinIO{
		Client:     imgClient,
		SignClient: imgSignClient,
		Bucket:     imgBucket,
	}, &MinIO{
		Client:     fileClient,
		SignClient: fileSignClient,
		Bucket:     fileBucket,
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
	client := m.SignClient
	if client == nil {
		client = m.Client
	}
	url, err := client.PresignedGetObject(
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
