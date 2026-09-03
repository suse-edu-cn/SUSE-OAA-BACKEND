package handler

import (
	"suseoaa/internal/request"
	"suseoaa/internal/service"
	"suseoaa/pkg/response"

	"github.com/gin-gonic/gin"
)

type FileHandler struct {
	FileService service.FileService
}

func NewFileHandler(s service.FileService) FileHandler {
	return FileHandler{FileService: s}
}

func (f *FileHandler) UploadImage(c *gin.Context) {
	var req request.UploadImageReq
	if err := c.ShouldBind(&req); err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	ctx := c.Request.Context()
	res, err := f.FileService.UploadImage(ctx, req.File, req.Scene)
	if err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.Success(c, res)
	return
}

func (f *FileHandler) UploadFile(c *gin.Context) {
	var req request.UploadFileReq
	if err := c.ShouldBind(&req); err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	ctx := c.Request.Context()
	res, err := f.FileService.UploadFile(ctx, req.File, req.Scene)
	if err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.Success(c, res)
	return
}
