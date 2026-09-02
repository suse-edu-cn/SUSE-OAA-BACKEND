package service

import "suseoaa/internal/storage"

type FileService struct {
	Storage *storage.MinIO
}

func NewFileService(storage *storage.MinIO) FileService {
	return FileService{
		Storage: storage,
	}
}

func (f *FileService) UploadImage() {
	
}
