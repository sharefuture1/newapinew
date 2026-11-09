package controller

// Wending OAuth: first-party OAuth 2.0 authorization server (authorization
// code + PKCE S256) that hands a dedicated new-api API key to registered
// first-party products. Fork-local customization; see WENDING_PATCHES.md and
// docs/WENDING_OAUTH.md.
//
// Security references: RFC 6749, RFC 7636 (PKCE), RFC 9700 (OAuth 2.0 Security
// BCP), OWASP ASVS 5.0 V10 (OAuth and OIDC).

import (
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/gin-gonic/gin"
)

// wendingOAuthAuthorizeParams are the RFC 6749 §4.1.1 request parameters.
type wendingOAuthAuthorizeParams struct {
	ResponseType        string `json:"response_type" form:"response_type"`
	ClientId            string `json:"client_id" form:"client_id"`
	RedirectUri         string `json:"redirect_uri" form:"redirect_uri"`
	State               string `json:"state" form:"state"`
	Scope               string `json:"scope" form:"scope"`
	CodeChallenge       string `json:"code_challenge" form:"code_challenge"`
	CodeChallengeMethod string `json:"code_challenge_method" form:"code_challenge_method"`
}

type wendingOAuthApproveRequest struct {
	wendingOAuthAuthorizeParams
	Approve        bool   `json:"approve"`
	UnlimitedQuota bool   `json:"unlimited_quota"`
	RemainQuota    int    `json:"remain_quota"`
	Group          string `json:"group"`
}

// wendingOAuthRequestError is an error in the authorization request.
// RedirectTrusted reports whether the client and redirect_uri were verified,
// i.e. whether the error may be sent back to redirect_uri (RFC 6749 §4.1.2.1).
type wendingOAuthRequestError struct {
	Code            string
	Description     string
	RedirectTrusted bool
}

func (e *wendingOAuthRequestError) Error() string { return e.Description }

// validateWendingAuthorizeRequest validates the client, redirect_uri, and PKCE
// parameters server-side before any consent screen is shown.
func validateWendingAuthorizeRequest(params wendingOAuthAuthorizeParams) (*model.WendingOAuthClient, string, *wendingOAuthRequestError) {
	client, err := model.GetWendingOAuthClientByClientId(params.ClientId)
	if err != nil || !client.Enabled {
		return nil, "", &wendingOAuthRequestError{Code: "invalid_client", Description: "Unknown or disabled client"}
	}
	if !client.HasRedirectURI(params.RedirectUri) {
		return client, "", &wendingOAuthRequestError{Code: "invalid_request", Description: "redirect_uri is not registered for this client"}
	}
	if params.ResponseType != "code" {
		return client, "", &wendingOAuthRequestError{Code: "unsupported_response_type", Description: "response_type must be code", RedirectTrusted: true}
	}
	if params.CodeChallengeMethod != "S256" {
		return client, "", &wendingOAuthRequestError{Code: "invalid_request", Description: "code_challenge_method must be S256", RedirectTrusted: true}
	}
	if !isWendingPKCEChallenge(params.CodeChallenge) {
		return client, "", &wendingOAuthRequestError{Code: "invalid_request", Description: "code_challenge is missing or malformed", RedirectTrusted: true}
	}
	if len(params.State) > 1024 {
		return client, "", &wendingOAuthRequestError{Code: "invalid_request", Description: "state is too long", RedirectTrusted: true}
	}
	scope := strings.TrimSpace(params.Scope)
	if scope == "" {
		scope = model.WendingOAuthDefaultScope
	}
	for item := range strings.FieldsSeq(scope) {
		switch item {
		case model.WendingOAuthScopeAPI, model.WendingOAuthScopeProfile, model.WendingOAuthScopeLogin, model.WendingOAuthScopeEmail:
		default:
			return client, "", &wendingOAuthRequestError{Code: "invalid_scope", Description: "Unsupported scope: " + item, RedirectTrusted: true}
		}
	}
	hasLogin := model.WendingOAuthScopeHas(scope, model.WendingOAuthScopeLogin)
	if hasLogin && model.WendingOAuthScopeHas(scope, model.WendingOAuthScopeAPI) {
		return client, "", &wendingOAuthRequestError{Code: "invalid_scope", Description: "login cannot be combined with api in one request", RedirectTrusted: true}
	}
	if model.WendingOAuthScopeHas(scope, model.WendingOAuthScopeEmail) && !hasLogin {
		return client, "", &wendingOAuthRequestError{Code: "invalid_scope", Description: "email requires the login scope", RedirectTrusted: true}
	}
	return client, model.WendingOAuthMergeScopes(scope, ""), nil
}

// isWendingPKCEChallenge accepts base64url(SHA-256) challenges: exactly 43 chars.
func isWendingPKCEChallenge(challenge string) bool {
	if len(challenge) != 43 {
		return false
	}
	decoded, err := base64.RawURLEncoding.DecodeString(challenge)
	return err == nil && len(decoded) == 32
}

func wendingOAuthRedirect(redirectURI string, values url.Values) string {
	parsed, err := url.Parse(redirectURI)
	if err != nil {
		return ""
	}
	query := parsed.Query()
	for key, items := range values {
		for _, item := range items {
			query.Set(key, item)
		}
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func wendingOAuthClientPublicView(client *model.WendingOAuthClient) gin.H {
	return gin.H{
		"client_id":               client.ClientId,
		"name":                    client.Name,
		"description":             client.Description,
		"icon_url":                client.IconUrl,
		"homepage_url":            client.HomepageUrl,
		"default_unlimited_quota": client.DefaultUnlimitedQuota,
		"default_remain_quota":    client.DefaultRemainQuota,
		"allow_unlimited_quota":   client.AllowUnlimitedQuota,
		"allowed_group":           client.AllowedGroup,
		"model_limits_enabled":    client.ModelLimitsEnabled,
		"model_limits":            client.ModelLimits,
		"trusted":                 client.Trusted,
	}
}

func recordWendingOAuthAudit(c *gin.Context, userId int, action string, success bool, content string, params model.AuditFields) {
	role := c.GetInt("role")
	if role == 0 {
		role = common.RoleCommonUser
	}
	model.RecordAuditLog(c, model.AuditLog{
		UserId:    userId,
		ActorRole: role,
		Category:  model.AuditCategorySecurity,
		Action:    action,
		Success:   success,
		Content:   content,
		Other:     model.AuditOther{Op: &model.AuditOperation{Action: action, Params: params}},
	})
}

// GetWendingOAuthAuthorize validates an authorization request for the consent screen.
// GET /api/wending/oauth/authorize (session)
func GetWendingOAuthAuthorize(c *gin.Context) {
	var params wendingOAuthAuthorizeParams
	_ = c.ShouldBindQuery(&params)
	client, scope, reqErr := validateWendingAuthorizeRequest(params)
	if reqErr != nil {
		body := gin.H{
			"success":          false,
			"message":          reqErr.Description,
			"oauth_error":      reqErr.Code,
			"redirect_trusted": reqErr.RedirectTrusted,
		}
		if reqErr.RedirectTrusted {
			body["redirect_to"] = wendingOAuthRedirect(params.RedirectUri, url.Values{
				"error": {reqErr.Code}, "error_description": {reqErr.Description}, "state": {params.State},
			})
		}
		c.JSON(http.StatusOK, body)
		return
	}
	userId := c.GetInt("id")
	userGroup, err := getTokenRequestUserGroup(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	groups := make([]gin.H, 0)
	for name, desc := range service.GetUserUsableGroups(userGroup) {
		if name == "auto" || !service.IsUserSelectableGroup(userGroup, name) {
			continue
		}
		groups = append(groups, gin.H{"name": name, "desc": desc})
	}
	var existing gin.H
	if grant, err := model.GetActiveWendingOAuthGrant(userId, client.ClientId); err == nil && grant != nil {
		existing = gin.H{"id": grant.Id, "created_at": grant.CreatedAt, "scope": grant.Scope}
	}
	loginOnly := model.WendingOAuthIsLoginScope(scope)
	common.ApiSuccess(c, gin.H{
		"client":         wendingOAuthClientPublicView(client),
		"scope":          scope,
		"login_only":     loginOnly,
		"skip_consent":   loginOnly && client.Trusted,
		"user_group":     userGroup,
		"groups":         groups,
		"existing_grant": existing,
		"quota_per_unit": common.QuotaPerUnit,
		"max_quota":      maxTokenQuota(),
	})
}

// PostWendingOAuthAuthorize records the user's decision and returns the
// redirect target. Requires an interactive browser session (PATs rejected).
// POST /api/wending/oauth/authorize (session, Bearer header: not CSRF-able)
func PostWendingOAuthAuthorize(c *gin.Context) {
	if _, ok := middleware.GetSessionAuthIdentity(c); !ok {
		common.ApiErrorStatus(c, http.StatusForbidden, errors.New("authorization consent requires an interactive login session"))
		return
	}
	var req wendingOAuthApproveRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorT(c, "Invalid parameters")
		return
	}
	userId := c.GetInt("id")
	client, scope, reqErr := validateWendingAuthorizeRequest(req.wendingOAuthAuthorizeParams)
	if reqErr != nil {
		body := gin.H{"success": false, "message": reqErr.Description, "oauth_error": reqErr.Code, "redirect_trusted": reqErr.RedirectTrusted}
		if reqErr.RedirectTrusted {
			body["redirect_to"] = wendingOAuthRedirect(req.RedirectUri, url.Values{
				"error": {reqErr.Code}, "error_description": {reqErr.Description}, "state": {req.State},
			})
		}
		c.JSON(http.StatusOK, body)
		return
	}
	auditParams := model.AuditFields{"client_id": client.ClientId, "client_name": client.Name}
	if !req.Approve {
		recordWendingOAuthAudit(c, userId, "wending_oauth.deny", true, "Denied authorization for "+client.Name, auditParams)
		common.ApiSuccess(c, gin.H{"redirect_to": wendingOAuthRedirect(req.RedirectUri, url.Values{
			"error": {"access_denied"}, "error_description": {"The user denied the request"}, "state": {req.State},
		})})
		return
	}
	if model.WendingOAuthIsLoginScope(scope) {
		issueWendingOAuthLogin(c, userId, client, scope, req, auditParams)
		return
	}

	settings := model.WendingOAuthTokenSettings{
		UnlimitedQuota:     req.UnlimitedQuota,
		RemainQuota:        req.RemainQuota,
		ModelLimitsEnabled: client.ModelLimitsEnabled,
		ModelLimits:        client.ModelLimits,
		Scope:              scope,
	}
	if settings.UnlimitedQuota {
		if !client.AllowUnlimitedQuota {
			common.ApiErrorT(c, "This application does not allow unlimited quota")
			return
		}
		settings.RemainQuota = 0
	} else if settings.RemainQuota <= 0 || settings.RemainQuota > maxTokenQuota() {
		common.ApiErrorT(c, "Quota value exceeds valid range, maximum is {{max}}", map[string]any{"max": maxTokenQuota()})
		return
	}
	userGroup, err := getTokenRequestUserGroup(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	group := strings.TrimSpace(req.Group)
	if client.AllowedGroup != "" {
		group = client.AllowedGroup
	}
	if group != "" && group != userGroup && !service.IsUserSelectableGroup(userGroup, group) {
		common.ApiErrorT(c, "Group {{group}} is unavailable to your account", map[string]any{"group": group})
		return
	}
	settings.Group = group

	grant, rotated, err := model.IssueWendingOAuthGrant(userId, client, settings, operation_setting.GetMaxUserTokens())
	if err != nil {
		recordWendingOAuthAudit(c, userId, "wending_oauth.authorize", false, "Authorization failed for "+client.Name, auditParams)
		common.ApiError(c, err)
		return
	}
	code, err := model.CreateWendingOAuthCode(grant, req.RedirectUri, req.CodeChallenge)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	auditParams["grant_id"], auditParams["token_id"], auditParams["rotated"] = grant.Id, grant.TokenId, rotated
	recordWendingOAuthAudit(c, userId, "wending_oauth.authorize", true, "Authorized application "+client.Name, auditParams)
	common.ApiSuccess(c, gin.H{
		"redirect_to": wendingOAuthRedirect(req.RedirectUri, url.Values{"code": {code}, "state": {req.State}}),
		"rotated":     rotated,
	})
}

// issueWendingOAuthLogin completes a sign-in-only ("login" scope) approval:
// it records the grant without creating an API key and returns the code.
func issueWendingOAuthLogin(c *gin.Context, userId int, client *model.WendingOAuthClient, scope string, req wendingOAuthApproveRequest, auditParams model.AuditFields) {
	grant, existing, err := model.IssueWendingOAuthLoginGrant(userId, client, scope)
	if err != nil {
		recordWendingOAuthAudit(c, userId, "wending_oauth.login_authorize", false, "Sign-in authorization failed for "+client.Name, auditParams)
		common.ApiError(c, err)
		return
	}
	code, err := model.CreateWendingOAuthCodeWithScope(grant, req.RedirectUri, req.CodeChallenge, scope)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	auditParams["grant_id"], auditParams["scope"], auditParams["trusted"], auditParams["existing"] = grant.Id, scope, client.Trusted, existing
	recordWendingOAuthAudit(c, userId, "wending_oauth.login_authorize", true, "Signed in to "+client.Name, auditParams)
	common.ApiSuccess(c, gin.H{
		"redirect_to": wendingOAuthRedirect(req.RedirectUri, url.Values{"code": {code}, "state": {req.State}}),
		"rotated":     false,
	})
}

// ---------------------------------------------------------------------------
// Token endpoint (public)
// ---------------------------------------------------------------------------

func wendingOAuthTokenError(c *gin.Context, status int, code, description string) {
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	if status == http.StatusUnauthorized {
		c.Header("WWW-Authenticate", `Basic realm="oauth"`)
	}
	c.JSON(status, gin.H{"error": code, "error_description": description})
}

// PostWendingOAuthToken exchanges an authorization code for the API key.
// POST /api/oauth/token (public, rate-limited). Accepts form or JSON bodies and
// client_secret_post or client_secret_basic client authentication.
func PostWendingOAuthToken(c *gin.Context) {
	values := map[string]string{}
	if strings.HasPrefix(c.ContentType(), "application/json") {
		var body map[string]any
		if err := common.DecodeJson(c.Request.Body, &body); err != nil {
			wendingOAuthTokenError(c, http.StatusBadRequest, "invalid_request", "Malformed JSON body")
			return
		}
		for key, value := range body {
			if s, ok := value.(string); ok {
				values[key] = s
			}
		}
	} else {
		if err := c.Request.ParseForm(); err != nil {
			wendingOAuthTokenError(c, http.StatusBadRequest, "invalid_request", "Malformed form body")
			return
		}
		for key := range c.Request.PostForm {
			values[key] = c.Request.PostForm.Get(key)
		}
	}
	clientId, clientSecret := values["client_id"], values["client_secret"]
	if basicId, basicSecret, ok := c.Request.BasicAuth(); ok {
		decodedId, errId := url.QueryUnescape(basicId)
		decodedSecret, errSecret := url.QueryUnescape(basicSecret)
		if errId != nil || errSecret != nil || (clientId != "" && clientId != decodedId) {
			wendingOAuthTokenError(c, http.StatusUnauthorized, "invalid_client", "Client authentication failed")
			return
		}
		clientId, clientSecret = decodedId, decodedSecret
	}
	if values["grant_type"] != "authorization_code" {
		wendingOAuthTokenError(c, http.StatusBadRequest, "unsupported_grant_type", "grant_type must be authorization_code")
		return
	}
	client, err := model.GetWendingOAuthClientByClientId(clientId)
	if err != nil || !client.Enabled {
		wendingOAuthTokenError(c, http.StatusUnauthorized, "invalid_client", "Client authentication failed")
		return
	}
	if !client.IsPublic || clientSecret != "" {
		if !client.VerifySecret(clientSecret) {
			wendingOAuthTokenError(c, http.StatusUnauthorized, "invalid_client", "Client authentication failed")
			return
		}
	}
	code, verifier, redirectURI := values["code"], values["code_verifier"], values["redirect_uri"]
	if code == "" || verifier == "" || redirectURI == "" {
		wendingOAuthTokenError(c, http.StatusBadRequest, "invalid_request", "code, code_verifier and redirect_uri are required")
		return
	}
	row, err := model.GetWendingOAuthCode(code)
	if err != nil {
		wendingOAuthTokenError(c, http.StatusBadRequest, "invalid_grant", "Invalid authorization code")
		return
	}
	auditParams := model.AuditFields{"client_id": client.ClientId, "grant_id": row.GrantId}
	failGrant := func(description string) {
		recordWendingOAuthAudit(c, row.UserId, "wending_oauth.token_exchange", false, "Code exchange failed: "+description, auditParams)
		wendingOAuthTokenError(c, http.StatusBadRequest, "invalid_grant", description)
	}
	if row.ClientId != client.ClientId {
		failGrant("Authorization code was not issued to this client")
		return
	}
	if row.UsedAt != 0 {
		failGrant("Authorization code has already been used")
		return
	}
	if row.ExpiresAt < common.GetTimestamp() {
		failGrant("Authorization code has expired")
		return
	}
	if row.RedirectUri != redirectURI {
		failGrant("redirect_uri does not match the authorization request")
		return
	}
	if !model.VerifyWendingPKCES256(verifier, row.CodeChallenge) {
		failGrant("PKCE verification failed")
		return
	}
	// Mark the code used before releasing the key: a code is single-use even
	// when a later check fails.
	if err := model.ConsumeWendingOAuthCode(row); err != nil {
		failGrant("Authorization code has already been used")
		return
	}
	grant, err := model.GetWendingOAuthGrantById(row.GrantId)
	if err != nil || grant.RevokedAt != 0 || grant.UserId != row.UserId || grant.ClientId != client.ClientId {
		failGrant("Authorization grant has been revoked")
		return
	}
	if model.WendingOAuthIsLoginScope(row.Scope) {
		user, err := model.GetUserCache(grant.UserId)
		if err != nil || user.Status != common.UserStatusEnabled {
			failGrant("The authorizing user is unavailable")
			return
		}
		accessToken, expiresIn, err := model.CreateWendingOAuthAccessToken(grant, row.Scope)
		if err != nil {
			wendingOAuthTokenError(c, http.StatusInternalServerError, "server_error", "Failed to issue access token")
			return
		}
		model.TouchWendingOAuthGrant(grant.Id)
		recordWendingOAuthAudit(c, grant.UserId, "wending_oauth.token_exchange", true, "Issued sign-in token to "+client.Name, auditParams)
		c.Header("Cache-Control", "no-store")
		c.Header("Pragma", "no-cache")
		c.JSON(http.StatusOK, gin.H{
			"access_token": accessToken,
			"token_type":   "Bearer",
			"expires_in":   expiresIn,
			"scope":        row.Scope,
			"user_id":      grant.UserId,
		})
		return
	}
	token, err := model.GetTokenByIds(grant.TokenId, grant.UserId)
	if err != nil || token.Status != common.TokenStatusEnabled {
		failGrant("The API key for this authorization is no longer available")
		return
	}
	user, err := model.GetUserCache(grant.UserId)
	if err != nil || user.Status != common.UserStatusEnabled {
		failGrant("The authorizing user is unavailable")
		return
	}
	model.TouchWendingOAuthGrant(grant.Id)
	recordWendingOAuthAudit(c, grant.UserId, "wending_oauth.token_exchange", true, "Issued API key to "+client.Name, auditParams)
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	c.JSON(http.StatusOK, gin.H{
		"access_token": "sk-" + token.Key,
		"token_type":   "Bearer",
		"base_url":     strings.TrimSuffix(system_setting.ServerAddress, "/"),
		"scope":        grant.Scope,
		"user_id":      grant.UserId,
	})
}

// GetWendingOAuthUserInfo returns who is connected, authenticated by the
// OAuth-issued API key. GET /api/oauth/userinfo (Bearer sk-...)
func GetWendingOAuthUserInfo(c *gin.Context) {
	raw := strings.TrimSpace(c.GetHeader("Authorization"))
	if len(raw) > 7 && strings.EqualFold(raw[:7], "Bearer ") {
		raw = strings.TrimSpace(raw[7:])
	} else {
		raw = ""
	}
	key := strings.TrimPrefix(raw, "sk-")
	unauthorized := func() {
		c.Header("WWW-Authenticate", `Bearer error="invalid_token"`)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token", "error_description": "The access token is invalid or revoked"})
	}
	if key == "" {
		unauthorized()
		return
	}
	if strings.HasPrefix(raw, model.WendingOAuthAccessTokenPrefix) {
		getWendingOAuthLoginUserInfo(c, raw, unauthorized)
		return
	}
	token, err := model.GetWendingOAuthTokenByKey(key)
	if err != nil || token.Status != common.TokenStatusEnabled {
		unauthorized()
		return
	}
	grant, err := model.GetWendingOAuthGrantByTokenId(token.Id)
	if err != nil || grant == nil || grant.UserId != token.UserId {
		unauthorized()
		return
	}
	user, err := model.GetUserById(token.UserId, false)
	if err != nil || user.Status != common.UserStatusEnabled {
		unauthorized()
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{
		"sub":            strconv.Itoa(user.Id),
		"user_id":        user.Id,
		"username":       user.Username,
		"display_name":   user.DisplayName,
		"group":          user.Group,
		"client_id":      grant.ClientId,
		"scope":          grant.Scope,
		"quota_per_unit": common.QuotaPerUnit,
		"user_quota": gin.H{
			"remain_quota": user.Quota,
			"used_quota":   user.UsedQuota,
		},
		"token_quota": gin.H{
			"unlimited_quota": token.UnlimitedQuota,
			"remain_quota":    token.RemainQuota,
			"used_quota":      token.UsedQuota,
			"group":           token.Group,
		},
	})
}

// getWendingOAuthLoginUserInfo answers userinfo for a sign-in (wdat_) token.
// It returns identity only: no balance and no API key data.
func getWendingOAuthLoginUserInfo(c *gin.Context, raw string, unauthorized func()) {
	row, grant, err := model.ValidateWendingOAuthAccessToken(raw)
	if err != nil {
		unauthorized()
		return
	}
	user, err := model.GetUserById(row.UserId, false)
	if err != nil || user.Status != common.UserStatusEnabled {
		unauthorized()
		return
	}
	body := gin.H{
		"sub":          strconv.Itoa(user.Id),
		"user_id":      user.Id,
		"username":     user.Username,
		"display_name": user.DisplayName,
		"client_id":    grant.ClientId,
		"scope":        row.Scope,
	}
	// E-mail is released only when the "email" scope was granted. new-api does
	// not track whether an e-mail was verified, so relying parties must NOT use
	// it to merge accounts (email_verified=false).
	if model.WendingOAuthScopeHas(row.Scope, model.WendingOAuthScopeEmail) {
		body["email"] = user.Email
		body["email_verified"] = false
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, body)
}

// ---------------------------------------------------------------------------
// User grant management
// ---------------------------------------------------------------------------

// GET /api/wending/oauth/grants (session)
func ListSelfWendingOAuthGrants(c *gin.Context) {
	views, err := model.ListWendingOAuthGrantViews(c.GetInt("id"), false, 200)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, views)
}

// DELETE /api/wending/oauth/grants/:id (session)
func RevokeSelfWendingOAuthGrant(c *gin.Context) {
	userId := c.GetInt("id")
	id, _ := strconv.Atoi(c.Param("id"))
	grant, err := model.GetWendingOAuthGrantById(id)
	if err != nil || grant.UserId != userId {
		common.ApiErrorT(c, "Authorization not found")
		return
	}
	if err := model.RevokeWendingOAuthGrant(grant); err != nil {
		common.ApiError(c, err)
		return
	}
	recordWendingOAuthAudit(c, userId, "wending_oauth.revoke", true, "Revoked application authorization",
		model.AuditFields{"client_id": grant.ClientId, "grant_id": grant.Id, "token_id": grant.TokenId})
	common.ApiSuccess(c, nil)
}

// ---------------------------------------------------------------------------
// Admin: clients and grants
// ---------------------------------------------------------------------------

type wendingOAuthClientRequest struct {
	Name                  string   `json:"name"`
	Description           string   `json:"description"`
	IconUrl               string   `json:"icon_url"`
	HomepageUrl           string   `json:"homepage_url"`
	LaunchUrl             string   `json:"launch_url"`
	ShowInSidebar         bool     `json:"show_in_sidebar"`
	SidebarOrder          int      `json:"sidebar_order"`
	RedirectUris          []string `json:"redirect_uris"`
	IsPublic              bool     `json:"is_public"`
	Enabled               bool     `json:"enabled"`
	DefaultUnlimitedQuota bool     `json:"default_unlimited_quota"`
	DefaultRemainQuota    int      `json:"default_remain_quota"`
	AllowUnlimitedQuota   bool     `json:"allow_unlimited_quota"`
	AllowedGroup          string   `json:"allowed_group"`
	ModelLimitsEnabled    bool     `json:"model_limits_enabled"`
	ModelLimits           string   `json:"model_limits"`
	Trusted               bool     `json:"trusted"`
}

type wendingOAuthClientView struct {
	*model.WendingOAuthClient
	RedirectUris []string `json:"redirect_uris"`
	HasSecret    bool     `json:"has_secret"`
}

func newWendingOAuthClientView(client *model.WendingOAuthClient) wendingOAuthClientView {
	return wendingOAuthClientView{WendingOAuthClient: client, RedirectUris: client.GetRedirectURIs(), HasSecret: client.ClientSecretHash != ""}
}

func applyWendingOAuthClientRequest(client *model.WendingOAuthClient, req wendingOAuthClientRequest) error {
	name := strings.TrimSpace(req.Name)
	if name == "" || len([]rune(name)) > 40 {
		return errors.New("name is required and must be at most 40 characters")
	}
	for _, link := range []string{req.IconUrl, req.HomepageUrl, req.LaunchUrl} {
		link = strings.TrimSpace(link)
		if link == "" {
			continue
		}
		parsed, err := url.Parse(link)
		if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" || len(link) > 1024 {
			return errors.New("icon_url, homepage_url and launch_url must be http(s) URLs")
		}
	}
	if req.ShowInSidebar && strings.TrimSpace(req.LaunchUrl) == "" {
		return errors.New("launch_url is required when show_in_sidebar is enabled")
	}
	if req.SidebarOrder < -1000000 || req.SidebarOrder > 1000000 {
		return errors.New("sidebar_order is out of range")
	}
	if err := client.SetRedirectURIs(req.RedirectUris); err != nil {
		return err
	}
	if req.DefaultRemainQuota < 0 || req.DefaultRemainQuota > maxTokenQuota() {
		return errors.New("default_remain_quota is out of range")
	}
	client.Name = name
	client.Description = strings.TrimSpace(req.Description)
	client.IconUrl = strings.TrimSpace(req.IconUrl)
	client.HomepageUrl = strings.TrimSpace(req.HomepageUrl)
	client.LaunchUrl = strings.TrimSpace(req.LaunchUrl)
	client.ShowInSidebar = req.ShowInSidebar
	client.SidebarOrder = req.SidebarOrder
	client.IsPublic = req.IsPublic
	client.Enabled = req.Enabled
	client.DefaultUnlimitedQuota = req.DefaultUnlimitedQuota && req.AllowUnlimitedQuota
	client.DefaultRemainQuota = req.DefaultRemainQuota
	client.AllowUnlimitedQuota = req.AllowUnlimitedQuota
	client.AllowedGroup = strings.TrimSpace(req.AllowedGroup)
	client.ModelLimitsEnabled = req.ModelLimitsEnabled
	client.ModelLimits = strings.TrimSpace(req.ModelLimits)
	client.Trusted = req.Trusted
	return nil
}

// GET /api/wending/oauth/sidebar-apps (any logged-in user)
// Returns the enabled apps the admin marked for the console left sidebar.
func ListWendingOAuthSidebarApps(c *gin.Context) {
	clients, err := model.ListWendingOAuthSidebarApps()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	apps := make([]gin.H, 0, len(clients))
	for _, client := range clients {
		apps = append(apps, gin.H{
			"client_id":     client.ClientId,
			"name":          client.Name,
			"description":   client.Description,
			"icon_url":      client.IconUrl,
			"homepage_url":  client.HomepageUrl,
			"launch_url":    client.LaunchUrl,
			"sidebar_order": client.SidebarOrder,
		})
	}
	common.ApiSuccess(c, apps)
}

// GET /api/wending/oauth/clients (root)
func ListWendingOAuthClients(c *gin.Context) {
	clients, err := model.ListWendingOAuthClients()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	views := make([]wendingOAuthClientView, 0, len(clients))
	for _, client := range clients {
		views = append(views, newWendingOAuthClientView(client))
	}
	common.ApiSuccess(c, views)
}

// POST /api/wending/oauth/clients (root). The secret is returned only once.
func CreateWendingOAuthClient(c *gin.Context) {
	var req wendingOAuthClientRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorT(c, "Invalid parameters")
		return
	}
	client := &model.WendingOAuthClient{}
	if err := applyWendingOAuthClientRequest(client, req); err != nil {
		common.ApiError(c, err)
		return
	}
	clientId, err := model.WendingOAuthRandomToken(model.WendingOAuthClientIdPrefix)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	client.ClientId = clientId[:len(model.WendingOAuthClientIdPrefix)+24]
	secret, err := client.RotateSecret()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := client.Insert(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"client": newWendingOAuthClientView(client), "client_secret": secret})
}

// PUT /api/wending/oauth/clients/:id (root)
func UpdateWendingOAuthClient(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	client, err := model.GetWendingOAuthClientById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	var req wendingOAuthClientRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorT(c, "Invalid parameters")
		return
	}
	if err := applyWendingOAuthClientRequest(client, req); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := client.Save(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, newWendingOAuthClientView(client))
}

// POST /api/wending/oauth/clients/:id/secret (root). Rotates and returns the new secret once.
func RotateWendingOAuthClientSecret(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	client, err := model.GetWendingOAuthClientById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	secret, err := client.RotateSecret()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := client.Save(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"client_secret": secret})
}

// DELETE /api/wending/oauth/clients/:id (root). Revokes every grant of the client.
func DeleteWendingOAuthClient(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	client, err := model.GetWendingOAuthClientById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.DeleteWendingOAuthClient(client); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

// GET /api/wending/oauth/admin/grants?user_id=&include_revoked= (admin)
func ListAdminWendingOAuthGrants(c *gin.Context) {
	userId, _ := strconv.Atoi(c.Query("user_id"))
	views, err := model.ListWendingOAuthGrantViews(userId, c.Query("include_revoked") == "true", 500)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, views)
}

// DELETE /api/wending/oauth/admin/grants/:id (admin)
func RevokeAdminWendingOAuthGrant(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	grant, err := model.GetWendingOAuthGrantById(id)
	if err != nil {
		common.ApiErrorT(c, "Authorization not found")
		return
	}
	if err := model.RevokeWendingOAuthGrant(grant); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}
