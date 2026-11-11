package handler

import (
	"net/http"
	"net/url"
	"strings"
	"suseoaa/internal/request"
	"suseoaa/internal/service"
	"suseoaa/pkg/response"
	"suseoaa/pkg/utils"

	"github.com/gin-gonic/gin"
)

type OAuthHandler struct {
	OAuthService *service.OAuthService
	JwtSecret    string
}

func NewOAuthHandler(oAuthService *service.OAuthService, jwtSecret string) *OAuthHandler {
	return &OAuthHandler{
		OAuthService: oAuthService,
		JwtSecret:    jwtSecret,
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
func (o *OAuthHandler) resolveUserID(c *gin.Context) uint64 {
	token, _ := c.Cookie("token")
	if token == "" {
		token = c.Query("token")
	}
	if token != "" {
		if claims, err := utils.ParseToken(token, o.JwtSecret); err == nil {
			return claims.UserID
		}
	}
	if o.OAuthService != nil && o.OAuthService.UserRepo != nil {
		if refreshToken, _ := c.Cookie("refresh_token"); refreshToken != "" {
			ctx := c.Request.Context()
			record, err := o.OAuthService.UserRepo.GetRefreshTokenByToken(ctx, refreshToken)
			if err == nil && record != nil {
				user, uErr := o.OAuthService.UserRepo.FindUserById(ctx, record.UserID)
				if uErr == nil {
					newToken, gErr := utils.GenerateToken(user.Name, user.ID, user.StudentID, o.JwtSecret, 15)
					if gErr == nil {
						c.SetCookie("token", newToken, 15*60, "/", "", false, false)
						return user.ID
					}
				}
			}
		}
	}
	return 0
}

func (o *OAuthHandler) Authorize(c *gin.Context) {
	var req request.AuthorizeReq
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":             "invalid_request",
			"error_description": err.Error(),
		})
		return
	}
	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":             "invalid_request",
			"error_description": err.Error(),
		})
		return
	}

	userID := o.resolveUserID(c)
	if userID == 0 {
		loginURL := "https://suseoaa.com/auth"
		c.Redirect(http.StatusFound, loginURL+"?redirect="+url.QueryEscape(c.Request.RequestURI))
		return
	}

	ctx := c.Request.Context()
	redirectURL, err := o.OAuthService.Authorize(ctx, userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":             "invalid_request",
			"error_description": err.Error(),
		})
		return
	}
	c.Redirect(http.StatusFound, redirectURL)
	return
}
func (o *OAuthHandler) Token(c *gin.Context) {
	var req request.TokenReq
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":             "invalid_request",
			"error_description": err.Error(),
		})
		return
	}
	if req.ClientID == "" || req.ClientSecret == "" {
		if u, p, ok := c.Request.BasicAuth(); ok {
			if req.ClientID == "" {
				req.ClientID = u
			}
			if req.ClientSecret == "" {
				req.ClientSecret = p
			}
		}
	}
	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":             "invalid_request",
			"error_description": err.Error(),
		})
		return
	}
	ctx := c.Request.Context()
	res, err := o.OAuthService.ExchangeCode(ctx, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":             "invalid_grant",
			"error_description": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, res)
	return
}

func (o *OAuthHandler) JWKS(c *gin.Context) {
	c.JSON(http.StatusOK, o.OAuthService.OIDCManager.GetJWKS())
}

func (o *OAuthHandler) OpenIDConfiguration(c *gin.Context) {
	scheme := "http"
	if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	fallbackBaseURL := scheme + "://" + c.Request.Host
	c.JSON(http.StatusOK, o.OAuthService.OIDCManager.GetOpenIDConfiguration(fallbackBaseURL))
}
func (o *OAuthHandler) UserInfo(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	var token string
	if len(authHeader) > 7 && strings.EqualFold(authHeader[:7], "bearer ") {
		token = strings.TrimSpace(authHeader[7:])
	}
	if token == "" {
		c.Header("WWW-Authenticate", `Bearer error="invalid_token", error_description="missing or invalid bearer token"`)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":             "invalid_token",
			"error_description": "缺少或不合法的 Bearer Token",
		})
		return
	}
	ctx := c.Request.Context()
	res, err := o.OAuthService.GetUserInfo(ctx, token)
	if err != nil {
		c.Header("WWW-Authenticate", `Bearer error="invalid_token", error_description="`+err.Error()+`"`)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":             "invalid_token",
			"error_description": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, res)
	return
}
