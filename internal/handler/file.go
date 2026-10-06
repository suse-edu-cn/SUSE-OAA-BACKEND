package handler

import (
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
		if err.Error() == "图片体积过大" || err.Error() == "图片类型错误" {
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
		if err.Error() == "文件体积过大" {
			response.BadRequest(c, err.Error())
			return
		}
		response.ServerError(c, "上传文件", err)
		return
	}
	response.Success(c, res)
}
