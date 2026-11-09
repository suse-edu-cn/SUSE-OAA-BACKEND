package handler

import (
	"suseoaa/internal/request"
	"suseoaa/internal/service"
	"suseoaa/pkg/response"

	"github.com/gin-gonic/gin"
)

type OAuthHandler struct {
	OAuthService *service.OAuthService
}

func NewOAuthHandler(oAuthService *service.OAuthService) *OAuthHandler {
	return &OAuthHandler{
		OAuthService: oAuthService,
	}
}

func (o *OAuthHandler) CreateOAuthClient(c *gin.Context) {
	var req request.CreateOAuthClientReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err)
		return
	}
	if err := req.Validate(); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	ctx := c.Request.Context()
	id := c.GetUint64("user_id")
	res, err := o.OAuthService.CreateOAuthClient(ctx, req, id)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, res)
	return
}
func (o *OAuthHandler) UpdateOAuthClient(c *gin.Context) {
	var req request.UpdateOAuthClientReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err)
		return
	}
	if err := req.Validate(); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	ctx := c.Request.Context()
	id := c.GetUint64("user_id")
	err := o.OAuthService.UpdateOAuthClient(ctx, id, req)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, nil)
	return
}
func (o *OAuthHandler) GetOAuthClient(c *gin.Context) {
	var req request.GetOAuthClientReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.ParamError(c, err)
		return
	}
	ctx := c.Request.Context()
	id := c.GetUint64("user_id")
	res, err := o.OAuthService.GetOAuthClientByID(ctx, req.ClientID, id)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, res)
	return

}
func (o *OAuthHandler) GetOAuthClientList(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.GetUint64("user_id")
	res, err := o.OAuthService.GetOAuthClientList(ctx, id)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, res)
	return
}
func (o *OAuthHandler) DeleteOAuthClient(c *gin.Context) {
	var req request.DeleteOAuthClientReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err)
		return
	}
	ctx := c.Request.Context()
	id := c.GetUint64("user_id")
	err := o.OAuthService.DeleteOAuthClient(ctx, req.ClientID, id)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, nil)
	return
}
func (o *OAuthHandler) ResetOAuthClientSecret(c *gin.Context) {
	var req request.ResetOAuthSecretReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err)
		return
	}
	ctx := c.Request.Context()
	id := c.GetUint64("user_id")
	secret, err := o.OAuthService.ResetOAuthClientSecret(ctx, req.ClientID, id)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, secret)
	return
}
func (o *OAuthHandler) Authorize(c *gin.Context) {
	var req request.AuthorizeReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.ParamError(c, err)
		return
	}
	if err := req.Validate(); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	ctx := c.Request.Context()
	userID := c.GetUint64("user_id")

	redirectURL, err := o.OAuthService.Authorize(ctx, userID, req)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, redirectURL)
	return
}
