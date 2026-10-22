package handler

import (
	"errors"
	"suseoaa/internal/request"
	"suseoaa/internal/service"
	"suseoaa/pkg/response"

	"github.com/gin-gonic/gin"
)

type FileHandler struct {
	FileService *service.FileService
}

func NewFileHandler(s *service.FileService) *FileHandler {
	return &FileHandler{FileService: s}
}

func (f *FileHandler) UploadImage(c *gin.Context) {
	var req request.UploadImageReq
	if err := c.ShouldBind(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	ctx := c.Request.Context()
	res, err := f.FileService.UploadImage(ctx, req.File, req.Scene)
	if err != nil {
		if errors.Is(err, service.ErrInvalidScene) ||
			errors.Is(err, service.ErrImageTooLarge) ||
			errors.Is(err, service.ErrInvalidImageType) {
			response.BadRequest(c, err.Error())
			return
		}
		response.ServerError(c, "上传图片", err)
		return
	}
	response.Success(c, res)
}

func (f *FileHandler) UploadFile(c *gin.Context) {
	var req request.UploadFileReq
	if err := c.ShouldBind(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	ctx := c.Request.Context()
	res, err := f.FileService.UploadFile(ctx, req.File, req.Scene)
	if err != nil {
		if errors.Is(err, service.ErrInvalidScene) ||
			errors.Is(err, service.ErrFileTooLarge) {
			response.BadRequest(c, err.Error())
			return
		}
		response.ServerError(c, "上传文件", err)
		return
	}
	response.Success(c, res)
}

