package request

import "mime/multipart"

type UploadImageReq struct {
	Scene string                `form:"scene" binding:"required"`
	File  *multipart.FileHeader `form:"file" binding:"required"`
}
type UploadFileReq struct {
	Scene string                `form:"scene" binding:"required"`
	File  *multipart.FileHeader `form:"file" binding:"required"`
}
