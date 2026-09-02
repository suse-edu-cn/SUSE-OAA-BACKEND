package handler

import (
	"suseoaa/internal/service"

	"github.com/gin-gonic/gin"
)

type FileHandler struct {
	FileService service.FileService
}

func NewFileHandler(s service.FileService) FileHandler {
	return FileHandler{FileService: s}
}

func (f *FileHandler) UploadImage(c *gin.Context) {}
