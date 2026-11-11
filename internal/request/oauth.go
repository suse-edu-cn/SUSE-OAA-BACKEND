package request

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
)

var SupportedScopes = map[string]bool{
	"openid":  true,
	"profile": true,
	"email":   true,
	"level":   true,
}
var clientIDRegex = regexp.MustCompile(`^[a-z0-9_-]{3,32}$`)

const DefaultScope = "openid profile email"

func validateAndCleanRedirectURIs(raw string) (string, error) {
	uris := strings.SplitSeq(raw, ",")
	var redirectURI []string
	for uri := range uris {
		u := strings.TrimSpace(uri)
		if u == "" {
			continue
		}
		parsedURL, err := url.ParseRequestURI(u)
		if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
			return "", errors.New("redirect_uris 中包含不合法的 URL: " + u)
		}
		redirectURI = append(redirectURI, u)
	}
	if len(redirectURI) == 0 {
		return "", errors.New("redirect_uris 不能为空")
	}
	return strings.Join(redirectURI, ","), nil
}

func validateAndCleanScope(raw string) (string, error) {
	if raw == "" {
		return DefaultScope, nil
	}
	scopes := strings.Fields(raw)
	for _, scope := range scopes {
		if !SupportedScopes[scope] {
			return "", errors.New("scope 中存在不合法权限：" + scope)
		}
	}
	return strings.Join(scopes, " "), nil
}

type CreateOAuthClientReq struct {
	Name         string `json:"name" binding:"required"`
	ClientID     string `json:"client_id" binding:"required"`
	RedirectURIs string `json:"redirect_uris" binding:"required"`
	Scope        string `json:"scope"`
}

func (c *CreateOAuthClientReq) Validate() error {
	c.Name = strings.TrimSpace(c.Name)
	c.ClientID = strings.TrimSpace(c.ClientID)
	if !clientIDRegex.MatchString(c.ClientID) {
		return errors.New("clientID格式不合法")
	}
	cleanedURIs, err := validateAndCleanRedirectURIs(c.RedirectURIs)
	if err != nil {
		return err
	}
	c.RedirectURIs = cleanedURIs
	cleanedScope, err := validateAndCleanScope(c.Scope)
	if err != nil {
		return err
	}
	c.Scope = cleanedScope
	return nil
}

type UpdateOAuthClientReq struct {
	ClientID     string `json:"client_id" binding:"required"`
	Name         string `json:"name" binding:"required"`
	RedirectURIs string `json:"redirect_uris" binding:"required"`
	Scope        string `json:"scope"`
	IsActive     *bool  `json:"is_active" binding:"required"`
}

func (u *UpdateOAuthClientReq) Validate() error {
	u.Name = strings.TrimSpace(u.Name)
	u.ClientID = strings.TrimSpace(u.ClientID)

	if !clientIDRegex.MatchString(u.ClientID) {
		return errors.New("clientID格式不合法")
	}

	cleanedURIs, err := validateAndCleanRedirectURIs(u.RedirectURIs)
	if err != nil {
		return err
	}
	u.RedirectURIs = cleanedURIs

	cleanedScope, err := validateAndCleanScope(u.Scope)
	if err != nil {
		return err
	}
	u.Scope = cleanedScope

	return nil
}

type GetOAuthClientReq struct {
	ClientID string `form:"client_id" json:"client_id" binding:"required"`
}

type DeleteOAuthClientReq struct {
	ClientID string `form:"client_id" json:"client_id" binding:"required"`
}
type ResetOAuthSecretReq struct {
	ClientID string `form:"client_id" json:"client_id" binding:"required"`
}
type AuthorizeReq struct {
	ClientID     string `form:"client_id" binding:"required"`
	RedirectURI  string `form:"redirect_uri" binding:"required"`
	ResponseType string `form:"response_type" binding:"required"`
	Scope        string `form:"scope"`
	State        string `form:"state"`
}

func (a *AuthorizeReq) Validate() error {
	a.ClientID = strings.TrimSpace(a.ClientID)
	a.RedirectURI = strings.TrimSpace(a.RedirectURI)
	a.ResponseType = strings.TrimSpace(a.ResponseType)
	a.Scope = strings.TrimSpace(a.Scope)
	a.State = strings.TrimSpace(a.State)

	if a.ResponseType != "code" {
		return errors.New("response_type 必须为 code")
	}

	return nil
}

type TokenReq struct {
	GrantType    string `form:"grant_type" json:"grant_type" binding:"required"`
	ClientID     string `form:"client_id" json:"client_id"`
	Code         string `form:"code" json:"code" binding:"required"`
	ClientSecret string `form:"client_secret" json:"client_secret"`
	RedirectURI  string `form:"redirect_uri" json:"redirect_uri" binding:"required"`
}

func (t *TokenReq) Validate() error {
	t.GrantType = strings.TrimSpace(t.GrantType)
	t.ClientID = strings.TrimSpace(t.ClientID)
	t.Code = strings.TrimSpace(t.Code)
	t.ClientSecret = strings.TrimSpace(t.ClientSecret)
	t.RedirectURI = strings.TrimSpace(t.RedirectURI)
	if t.GrantType != "authorization_code" {
		return errors.New("grant_type 错误，必须为 authorization_code")
	}
	if t.ClientID == "" {
		return errors.New("缺少 client_id")
	}
	if t.ClientSecret == "" {
		return errors.New("缺少 client_secret")
	}
	return nil
}

type TokenRes struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
}
type UserInfoRes struct {
	Sub               string `json:"sub"`
	Name              string `json:"name,omitempty"`
	PreferredUsername string `json:"preferred_username,omitempty"`
	StudentID         string `json:"student_id,omitempty"`
	Email             string `json:"email,omitempty"`
	Department        string `json:"department,omitempty"`
	Role              string `json:"role,omitempty"`
	Level             uint64 `json:"level,omitempty"`
}
