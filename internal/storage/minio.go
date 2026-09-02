package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var MaxFileSize int64
var MaxImageSize int64

type MinIO struct {
	Client *minio.Client
	Bucket string
}

func NewMinIO(
	endpoint string,
	accessKey string,
	secretKey string,
	useSSL bool,
	bucket string,
	maxFileSize int64,
	maxImageSize int64,
) *MinIO {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		panic(err)
	}
	MaxFileSize = maxFileSize
	MaxImageSize = maxImageSize
	return &MinIO{
		Client: client,
		Bucket: bucket,
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
