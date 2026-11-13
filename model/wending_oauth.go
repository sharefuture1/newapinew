package model

// Wending OAuth: first-party OAuth 2.0 authorization server ("问鼎AI账号一键授权").
// A registered client (product) asks a logged-in user for consent; on approval
// the user gets a dedicated API token (a normal tokens row, so billing, quota
// and limits behave exactly like any other key) that is handed to the client
// through the authorization-code + PKCE flow.
//
// This file is a fork-local customization; see WENDING_PATCHES.md.

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	WendingOAuthCodeTTLSeconds   = 300 // authorization codes live at most 5 minutes
	WendingOAuthDefaultScope     = "api"
	WendingOAuthClientIdPrefix   = "wd_"
	WendingOAuthSecretPrefix     = "wds_"
	WendingOAuthCodePrefix       = "wdc_"
	WendingOAuthTokenNamePrefix  = "授权:"
	wendingOAuthMaxRedirectURIs  = 20
	wendingOAuthMaxRedirectURILn = 1024
)

var (
	ErrWendingOAuthClientNotFound = errors.New("oauth client not found")
	ErrWendingOAuthCodeInvalid    = errors.New("authorization code is invalid")
	ErrWendingOAuthCodeExpired    = errors.New("authorization code has expired")
	ErrWendingOAuthCodeUsed       = errors.New("authorization code has already been used")
	ErrWendingOAuthGrantRevoked   = errors.New("authorization grant has been revoked")
)

// WendingOAuthClient is a registered application ("接入应用").
type WendingOAuthClient struct {
	Id               int    `json:"id"`
	ClientId         string `json:"client_id" gorm:"type:varchar(64);uniqueIndex"`
	ClientSecretHash string `json:"-" gorm:"type:varchar(128)"`
	Name             string `json:"name" gorm:"type:varchar(64)"`
	Description      string `json:"description" gorm:"type:text"`
	IconUrl          string `json:"icon_url" gorm:"type:varchar(1024)"`
	HomepageUrl      string `json:"homepage_url" gorm:"type:varchar(1024)"`
	// LaunchUrl is the app URL opened from the console sidebar (RP initiates OAuth/PKCE there).
	LaunchUrl string `json:"launch_url" gorm:"type:varchar(1024)"`
	// ShowInSidebar lists the app under "问鼎应用" for logged-in users when LaunchUrl is set.
	ShowInSidebar bool `json:"show_in_sidebar"`
	// SidebarOrder sorts sidebar apps ascending (ties broken by id ascending).
	SidebarOrder int `json:"sidebar_order"`
	// RedirectUris is a JSON array of exact-match redirect URIs.
	RedirectUris string `json:"-" gorm:"type:text"`
	// IsPublic marks a PKCE-only client that cannot keep a secret (SPA, CLI, MCP).
	IsPublic bool `json:"is_public"`
	Enabled  bool `json:"enabled"`
	// Default token settings proposed on the consent screen.
	DefaultUnlimitedQuota bool   `json:"default_unlimited_quota"`
	DefaultRemainQuota    int    `json:"default_remain_quota"`
	AllowUnlimitedQuota   bool   `json:"allow_unlimited_quota"`
	AllowedGroup          string `json:"allowed_group" gorm:"type:varchar(64)"` // empty: user's own group / user choice
	ModelLimitsEnabled    bool   `json:"model_limits_enabled"`
	ModelLimits           string `json:"model_limits" gorm:"type:text"`
	CreatedAt             int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt             int64  `json:"updated_at" gorm:"bigint"`
}

func (WendingOAuthClient) TableName() string { return "wending_oauth_clients" }

// WendingOAuthGrant records a user's consent for one client and the token it owns.
type WendingOAuthGrant struct {
	Id         int    `json:"id"`
	UserId     int    `json:"user_id" gorm:"index"`
	ClientId   string `json:"client_id" gorm:"type:varchar(64);index"`
	TokenId    int    `json:"token_id" gorm:"index"`
	Scope      string `json:"scope" gorm:"type:varchar(255)"`
	CreatedAt  int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt  int64  `json:"updated_at" gorm:"bigint"`
	LastUsedAt int64  `json:"last_used_at" gorm:"bigint;default:0"` // last successful code exchange
	RevokedAt  int64  `json:"revoked_at" gorm:"bigint;default:0;index"`
	// DeviceId / DeviceName are set by native clients (desktop apps) that run on
	// several machines: each device gets its own grant and key, so authorizing a
	// second computer does not rotate (and break) the key of the first one.
	// Empty for web clients (one grant per user + client, unchanged behaviour).
	DeviceId   string `json:"device_id" gorm:"type:varchar(64);default:'';index"`
	DeviceName string `json:"device_name" gorm:"type:varchar(64);default:''"`
}

func (WendingOAuthGrant) TableName() string { return "wending_oauth_grants" }

// WendingOAuthCode is a one-time authorization code. Only its SHA-256 is stored.
type WendingOAuthCode struct {
	Id            int    `json:"id"`
	CodeHash      string `json:"-" gorm:"type:varchar(64);uniqueIndex"`
	ClientId      string `json:"client_id" gorm:"type:varchar(64);index"`
	UserId        int    `json:"user_id" gorm:"index"`
	GrantId       int    `json:"grant_id" gorm:"index"`
	RedirectUri   string `json:"redirect_uri" gorm:"type:text"`
	CodeChallenge string `json:"-" gorm:"type:varchar(128)"`
	Scope         string `json:"scope" gorm:"type:varchar(255)"`
	ExpiresAt     int64  `json:"expires_at" gorm:"bigint;index"`
	UsedAt        int64  `json:"used_at" gorm:"bigint;default:0"`
	CreatedAt     int64  `json:"created_at" gorm:"bigint"`
}

func (WendingOAuthCode) TableName() string { return "wending_oauth_codes" }

// migrateWendingOAuth is the single migration hook called from migrateDB.
func migrateWendingOAuth() error {
	return DB.AutoMigrate(&WendingOAuthClient{}, &WendingOAuthGrant{}, &WendingOAuthCode{})
}

// ---------------------------------------------------------------------------
// Secrets and hashing
// ---------------------------------------------------------------------------

// WendingOAuthRandomToken returns prefix + base64url(32 random bytes).
func WendingOAuthRandomToken(prefix string) (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return prefix + base64.RawURLEncoding.EncodeToString(buf), nil
}

func WendingOAuthHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

// VerifyWendingPKCES256 checks code_verifier against an S256 code_challenge (RFC 7636 §4.6).
func VerifyWendingPKCES256(verifier, challenge string) bool {
	// RFC 7636 §4.1: 43..128 chars of [A-Z a-z 0-9 - . _ ~]
	if len(verifier) < 43 || len(verifier) > 128 {
		return false
	}
	for _, r := range verifier {
		if !(r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' || r == '.' || r == '_' || r == '~') {
			return false
		}
	}
	sum := sha256.Sum256([]byte(verifier))
	computed := base64.RawURLEncoding.EncodeToString(sum[:])
	return subtle.ConstantTimeCompare([]byte(computed), []byte(challenge)) == 1
}

// ---------------------------------------------------------------------------
// Clients
// ---------------------------------------------------------------------------

func (client *WendingOAuthClient) GetRedirectURIs() []string {
	if client.RedirectUris == "" {
		return []string{}
	}
	var uris []string
	if err := common.UnmarshalJsonStr(client.RedirectUris, &uris); err != nil {
		return []string{}
	}
	return uris
}

// SetRedirectURIs validates and stores redirect URIs. Each must be an absolute
// URI without a fragment; http is only allowed for loopback hosts.
func (client *WendingOAuthClient) SetRedirectURIs(uris []string) error {
	cleaned := make([]string, 0, len(uris))
	for _, raw := range uris {
		uri := strings.TrimSpace(raw)
		if uri == "" {
			continue
		}
		if err := ValidateWendingRedirectURI(uri); err != nil {
			return err
		}
		if !slices.Contains(cleaned, uri) {
			cleaned = append(cleaned, uri)
		}
	}
	if len(cleaned) == 0 {
		return errors.New("at least one redirect URI is required")
	}
	if len(cleaned) > wendingOAuthMaxRedirectURIs {
		return fmt.Errorf("at most %d redirect URIs are allowed", wendingOAuthMaxRedirectURIs)
	}
	data, err := common.Marshal(cleaned)
	if err != nil {
		return err
	}
	client.RedirectUris = string(data)
	return nil
}

// HasRedirectURI performs the exact string match required by OAuth 2.1 / RFC 9700.
func (client *WendingOAuthClient) HasRedirectURI(uri string) bool {
	return uri != "" && slices.Contains(client.GetRedirectURIs(), uri)
}

// MatchRedirectURI is HasRedirectURI plus the RFC 8252 §7.3 loopback rule for
// public (native) clients: a registered http://127.0.0.1/<path> or
// http://[::1]/<path> redirect URI matches the same URI on any port, because
// desktop apps listen on an ephemeral port chosen at runtime. Scheme, IP
// literal, path and query must still match exactly; "localhost" is not
// port-flexible (RFC 8252 §8.3 recommends the IP literal).
func (client *WendingOAuthClient) MatchRedirectURI(uri string) bool {
	if client.HasRedirectURI(uri) {
		return true
	}
	if !client.IsPublic || uri == "" {
		return false
	}
	want, ok := wendingLoopbackKey(uri)
	if !ok {
		return false
	}
	for _, registered := range client.GetRedirectURIs() {
		if got, ok := wendingLoopbackKey(registered); ok && got == want {
			return true
		}
	}
	return false
}

// wendingLoopbackKey returns the redirect URI without its port when it is an
// http URI on a loopback IP literal.
func wendingLoopbackKey(uri string) (string, bool) {
	parsed, err := url.Parse(uri)
	if err != nil || parsed.Scheme != "http" || parsed.User != nil || parsed.Fragment != "" {
		return "", false
	}
	host := parsed.Hostname()
	if host != "127.0.0.1" && host != "::1" {
		return "", false
	}
	if port := parsed.Port(); port != "" {
		n, err := strconv.Atoi(port)
		if err != nil || n < 1 || n > 65535 {
			return "", false
		}
	}
	path := parsed.EscapedPath()
	if path == "" {
		path = "/"
	}
	return "http://" + host + path + "?" + parsed.RawQuery, true
}

func (client *WendingOAuthClient) VerifySecret(secret string) bool {
	if secret == "" || client.ClientSecretHash == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(WendingOAuthHash(secret)), []byte(client.ClientSecretHash)) == 1
}

// RotateSecret sets a new random secret and returns the plaintext (shown once).
func (client *WendingOAuthClient) RotateSecret() (string, error) {
	secret, err := WendingOAuthRandomToken(WendingOAuthSecretPrefix)
	if err != nil {
		return "", err
	}
	client.ClientSecretHash = WendingOAuthHash(secret)
	return secret, nil
}

func ListWendingOAuthClients() ([]*WendingOAuthClient, error) {
	var clients []*WendingOAuthClient
	err := DB.Order("id desc").Find(&clients).Error
	return clients, err
}

// ListWendingOAuthSidebarApps returns enabled clients flagged for the console sidebar.
// Only clients with a non-empty launch_url are included. Order: sidebar_order ASC, id ASC.
func ListWendingOAuthSidebarApps() ([]*WendingOAuthClient, error) {
	var clients []*WendingOAuthClient
	err := DB.Where("enabled = ? AND show_in_sidebar = ? AND launch_url <> ?", true, true, "").
		Order("sidebar_order asc, id asc").Find(&clients).Error
	return clients, err
}

func GetWendingOAuthClientById(id int) (*WendingOAuthClient, error) {
	var client WendingOAuthClient
	if err := DB.Where("id = ?", id).First(&client).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrWendingOAuthClientNotFound
		}
		return nil, err
	}
	return &client, nil
}

func GetWendingOAuthClientByClientId(clientId string) (*WendingOAuthClient, error) {
	if clientId == "" {
		return nil, ErrWendingOAuthClientNotFound
	}
	var client WendingOAuthClient
	if err := DB.Where("client_id = ?", clientId).First(&client).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrWendingOAuthClientNotFound
		}
		return nil, err
	}
	return &client, nil
}

func (client *WendingOAuthClient) Insert() error {
	now := common.GetTimestamp()
	client.CreatedAt, client.UpdatedAt = now, now
	return DB.Create(client).Error
}

// Save persists every editable column, including zero values.
func (client *WendingOAuthClient) Save() error {
	client.UpdatedAt = common.GetTimestamp()
	return DB.Model(client).Select("name", "description", "icon_url", "homepage_url", "launch_url",
		"show_in_sidebar", "sidebar_order", "redirect_uris",
		"is_public", "enabled", "default_unlimited_quota", "default_remain_quota", "allow_unlimited_quota",
		"allowed_group", "model_limits_enabled", "model_limits", "client_secret_hash", "updated_at").
		Updates(client).Error
}

// DeleteWendingOAuthClient removes a client and revokes every grant (deleting their tokens).
func DeleteWendingOAuthClient(client *WendingOAuthClient) error {
	var grants []*WendingOAuthGrant
	if err := DB.Where("client_id = ? AND revoked_at = 0", client.ClientId).Find(&grants).Error; err != nil {
		return err
	}
	for _, grant := range grants {
		if err := RevokeWendingOAuthGrant(grant); err != nil {
			return err
		}
	}
	if err := DB.Where("client_id = ?", client.ClientId).Delete(&WendingOAuthCode{}).Error; err != nil {
		return err
	}
	return DB.Delete(client).Error
}

// ---------------------------------------------------------------------------
// Grants
// ---------------------------------------------------------------------------

func GetActiveWendingOAuthGrant(userId int, clientId string) (*WendingOAuthGrant, error) {
	return GetActiveWendingOAuthDeviceGrant(userId, clientId, "")
}

// GetActiveWendingOAuthDeviceGrant returns the active grant of one device
// (deviceId "" = web clients / clients that do not send a device id).
func GetActiveWendingOAuthDeviceGrant(userId int, clientId string, deviceId string) (*WendingOAuthGrant, error) {
	var grant WendingOAuthGrant
	err := DB.Where("user_id = ? AND client_id = ? AND device_id = ? AND revoked_at = 0", userId, clientId, deviceId).
		Order("id desc").First(&grant).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &grant, nil
}

func GetWendingOAuthGrantById(id int) (*WendingOAuthGrant, error) {
	var grant WendingOAuthGrant
	if err := DB.Where("id = ?", id).First(&grant).Error; err != nil {
		return nil, err
	}
	return &grant, nil
}

// WendingOAuthGrantView is a grant joined with client and token display data.
type WendingOAuthGrantView struct {
	WendingOAuthGrant
	ClientName     string `json:"client_name"`
	ClientIconUrl  string `json:"client_icon_url"`
	ClientHomepage string `json:"client_homepage_url"`
	TokenName      string `json:"token_name"`
	TokenStatus    int    `json:"token_status"`
	TokenExists    bool   `json:"token_exists"`
	UnlimitedQuota bool   `json:"unlimited_quota"`
	RemainQuota    int    `json:"remain_quota"`
	UsedQuota      int    `json:"used_quota"`
	Group          string `json:"group"`
}

// ListWendingOAuthGrantViews lists grants for one user (userId > 0) or all users
// (userId == 0). Revoked grants are included only when includeRevoked is set.
func ListWendingOAuthGrantViews(userId int, includeRevoked bool, limit int) ([]*WendingOAuthGrantView, error) {
	query := DB.Model(&WendingOAuthGrant{})
	if userId > 0 {
		query = query.Where("user_id = ?", userId)
	}
	if !includeRevoked {
		query = query.Where("revoked_at = 0")
	}
	if limit <= 0 || limit > 500 {
		limit = 500
	}
	var grants []*WendingOAuthGrant
	if err := query.Order("id desc").Limit(limit).Find(&grants).Error; err != nil {
		return nil, err
	}
	clientIds := make([]string, 0)
	tokenIds := make([]int, 0)
	for _, grant := range grants {
		if !slices.Contains(clientIds, grant.ClientId) {
			clientIds = append(clientIds, grant.ClientId)
		}
		tokenIds = append(tokenIds, grant.TokenId)
	}
	clients := map[string]*WendingOAuthClient{}
	if len(clientIds) > 0 {
		var rows []*WendingOAuthClient
		if err := DB.Where("client_id IN ?", clientIds).Find(&rows).Error; err != nil {
			return nil, err
		}
		for _, row := range rows {
			clients[row.ClientId] = row
		}
	}
	tokens := map[int]*Token{}
	if len(tokenIds) > 0 {
		var rows []*Token
		if err := DB.Where("id IN ?", tokenIds).Find(&rows).Error; err != nil {
			return nil, err
		}
		for _, row := range rows {
			tokens[row.Id] = row
		}
	}
	views := make([]*WendingOAuthGrantView, 0, len(grants))
	for _, grant := range grants {
		view := &WendingOAuthGrantView{WendingOAuthGrant: *grant}
		if client, ok := clients[grant.ClientId]; ok {
			view.ClientName, view.ClientIconUrl, view.ClientHomepage = client.Name, client.IconUrl, client.HomepageUrl
		}
		if token, ok := tokens[grant.TokenId]; ok {
			view.TokenExists = true
			view.TokenName, view.TokenStatus = token.Name, token.Status
			view.UnlimitedQuota, view.RemainQuota, view.UsedQuota, view.Group = token.UnlimitedQuota, token.RemainQuota, token.UsedQuota, token.Group
		}
		views = append(views, view)
	}
	return views, nil
}

// RevokeWendingOAuthGrant marks the grant revoked, deletes its token (so the key
// is rejected immediately, cache included) and invalidates outstanding codes.
func RevokeWendingOAuthGrant(grant *WendingOAuthGrant) error {
	if grant.RevokedAt == 0 {
		now := common.GetTimestamp()
		result := DB.Model(&WendingOAuthGrant{}).Where("id = ? AND revoked_at = 0", grant.Id).
			Updates(map[string]any{"revoked_at": now, "updated_at": now})
		if result.Error != nil {
			return result.Error
		}
		grant.RevokedAt = now
	}
	if err := DB.Model(&WendingOAuthCode{}).Where("grant_id = ? AND used_at = 0", grant.Id).
		Update("used_at", common.GetTimestamp()).Error; err != nil {
		return err
	}
	if grant.TokenId <= 0 {
		return nil
	}
	token, err := GetTokenByIds(grant.TokenId, grant.UserId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil // token already deleted by the user
		}
		return err
	}
	return token.Delete()
}

// WendingOAuthTokenSettings are the token limits chosen on the consent screen.
type WendingOAuthTokenSettings struct {
	UnlimitedQuota     bool
	RemainQuota        int
	Group              string
	ModelLimitsEnabled bool
	ModelLimits        string
	Scope              string
	// DeviceId / DeviceName: see WendingOAuthGrant. Validated by the controller.
	DeviceId   string
	DeviceName string
}

// IssueWendingOAuthGrant creates (or rotates) the user's grant for client and
// returns the grant. Policy: one active grant per (user, client). Re-approval
// keeps the same tokens row (usage history stays together) but generates a
// NEW key and applies the newly chosen limits, so any previously issued key
// for this app stops working.
func IssueWendingOAuthGrant(userId int, client *WendingOAuthClient, settings WendingOAuthTokenSettings, maxTokens int) (*WendingOAuthGrant, bool, error) {
	existing, err := GetActiveWendingOAuthDeviceGrant(userId, client.ClientId, settings.DeviceId)
	if err != nil {
		return nil, false, err
	}
	name := wendingOAuthTokenName(client.Name, settings.DeviceName)
	now := common.GetTimestamp()
	if existing != nil {
		token, err := GetTokenByIds(existing.TokenId, userId)
		if err == nil {
			newKey, err := common.GenerateKey()
			if err != nil {
				return nil, false, err
			}
			if cacheErr := invalidateTokenCacheForMutation(token.Key); cacheErr != nil {
				common.SysLog(common.LogText("failed to invalidate token cache before oauth rotation: %s", cacheErr.Error()))
			}
			err = DB.Model(&Token{}).Where("id = ?", token.Id).Updates(map[string]any{
				"key":                  newKey,
				"name":                 name,
				"status":               common.TokenStatusEnabled,
				"expired_time":         -1,
				"remain_quota":         settings.RemainQuota,
				"unlimited_quota":      settings.UnlimitedQuota,
				"group":                settings.Group,
				"model_limits_enabled": settings.ModelLimitsEnabled,
				"model_limits":         settings.ModelLimits,
			}).Error
			if err != nil {
				return nil, false, err
			}
			if err := DB.Model(&WendingOAuthGrant{}).Where("id = ?", existing.Id).
				Updates(map[string]any{"scope": settings.Scope, "updated_at": now, "device_name": settings.DeviceName}).Error; err != nil {
				return nil, false, err
			}
			existing.Scope, existing.UpdatedAt, existing.DeviceName = settings.Scope, now, settings.DeviceName
			return existing, true, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, err
		}
		// The user deleted the token manually: retire this grant and start over.
		if err := RevokeWendingOAuthGrant(existing); err != nil {
			return nil, false, err
		}
	}

	count, err := CountUserTokens(userId)
	if err != nil {
		return nil, false, err
	}
	if maxTokens > 0 && int(count) >= maxTokens {
		return nil, false, common.NewMessage("Maximum number of tokens reached ({{max}})", map[string]any{"max": maxTokens})
	}
	key, err := common.GenerateKey()
	if err != nil {
		return nil, false, err
	}
	token := &Token{
		UserId:              userId,
		Name:                name,
		Key:                 key,
		Status:              common.TokenStatusEnabled,
		CreatedTime:         now,
		AccessedTime:        now,
		ExpiredTime:         -1,
		RemainQuota:         settings.RemainQuota,
		UnlimitedQuota:      settings.UnlimitedQuota,
		ModelLimitsEnabled:  settings.ModelLimitsEnabled,
		ModelLimits:         settings.ModelLimits,
		Group:               settings.Group,
		AllowUserQuotaQuery: true,
	}
	grant := &WendingOAuthGrant{
		UserId:     userId,
		ClientId:   client.ClientId,
		Scope:      settings.Scope,
		CreatedAt:  now,
		UpdatedAt:  now,
		DeviceId:   settings.DeviceId,
		DeviceName: settings.DeviceName,
	}
	err = DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(token).Error; err != nil {
			return err
		}
		grant.TokenId = token.Id
		return tx.Create(grant).Error
	})
	if err != nil {
		return nil, false, err
	}
	return grant, false, nil
}

func wendingOAuthTokenName(appName string, deviceName string) string {
	name := []rune(WendingOAuthTokenNamePrefix + strings.TrimSpace(appName))
	if len(name) > 30 { // tokens.name is limited to 50 bytes by the token API; keep it short
		name = name[:30]
	}
	if device := strings.TrimSpace(deviceName); device != "" {
		name = append(name, []rune(" · "+device)...)
	}
	for len(string(name)) > 50 {
		name = name[:len(name)-1]
	}
	return strings.TrimSpace(string(name))
}

// ---------------------------------------------------------------------------
// Authorization codes
// ---------------------------------------------------------------------------

// CreateWendingOAuthCode stores a new one-time code and returns its plaintext.
func CreateWendingOAuthCode(grant *WendingOAuthGrant, redirectURI, codeChallenge string) (string, error) {
	code, err := WendingOAuthRandomToken(WendingOAuthCodePrefix)
	if err != nil {
		return "", err
	}
	now := common.GetTimestamp()
	row := &WendingOAuthCode{
		CodeHash:      WendingOAuthHash(code),
		ClientId:      grant.ClientId,
		UserId:        grant.UserId,
		GrantId:       grant.Id,
		RedirectUri:   redirectURI,
		CodeChallenge: codeChallenge,
		Scope:         grant.Scope,
		ExpiresAt:     now + WendingOAuthCodeTTLSeconds,
		CreatedAt:     now,
	}
	if err := DB.Create(row).Error; err != nil {
		return "", err
	}
	// Opportunistic cleanup of stale codes; failures are harmless.
	DB.Where("expires_at < ?", now-86400).Delete(&WendingOAuthCode{})
	return code, nil
}

func GetWendingOAuthCode(code string) (*WendingOAuthCode, error) {
	if !strings.HasPrefix(code, WendingOAuthCodePrefix) {
		return nil, ErrWendingOAuthCodeInvalid
	}
	var row WendingOAuthCode
	if err := DB.Where("code_hash = ?", WendingOAuthHash(code)).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrWendingOAuthCodeInvalid
		}
		return nil, err
	}
	return &row, nil
}

// ConsumeWendingOAuthCode atomically marks the code used. It returns
// ErrWendingOAuthCodeUsed if another request consumed it first.
func ConsumeWendingOAuthCode(row *WendingOAuthCode) error {
	now := common.GetTimestamp()
	if row.UsedAt != 0 {
		return ErrWendingOAuthCodeUsed
	}
	if row.ExpiresAt < now {
		return ErrWendingOAuthCodeExpired
	}
	result := DB.Model(&WendingOAuthCode{}).Where("id = ? AND used_at = 0", row.Id).Update("used_at", now)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrWendingOAuthCodeUsed
	}
	row.UsedAt = now
	return nil
}

func TouchWendingOAuthGrant(grantId int) {
	DB.Model(&WendingOAuthGrant{}).Where("id = ?", grantId).Update("last_used_at", common.GetTimestamp())
}

// GetWendingOAuthGrantByTokenId returns the active grant owning tokenId, or nil.
func GetWendingOAuthGrantByTokenId(tokenId int) (*WendingOAuthGrant, error) {
	var grant WendingOAuthGrant
	err := DB.Where("token_id = ? AND revoked_at = 0", tokenId).Order("id desc").First(&grant).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &grant, nil
}

// ValidateWendingRedirectURI accepts absolute https URIs, http URIs on loopback
// hosts (local development / native apps, RFC 8252 §7.3) and private-use
// custom schemes (e.g. com.example.app:/cb). Fragments are forbidden.
func ValidateWendingRedirectURI(uri string) error {
	if len(uri) > wendingOAuthMaxRedirectURILn {
		return errors.New("redirect URI is too long")
	}
	parsed, err := url.Parse(uri)
	if err != nil || parsed.Scheme == "" {
		return fmt.Errorf("invalid redirect URI: %s", uri)
	}
	if parsed.Fragment != "" || strings.Contains(uri, "#") {
		return fmt.Errorf("redirect URI must not contain a fragment: %s", uri)
	}
	switch strings.ToLower(parsed.Scheme) {
	case "https":
		if parsed.Host == "" {
			return fmt.Errorf("invalid redirect URI: %s", uri)
		}
	case "http":
		host := parsed.Hostname()
		if host != "localhost" && host != "127.0.0.1" && host != "::1" {
			return fmt.Errorf("http redirect URIs are only allowed for localhost: %s", uri)
		}
	case "javascript", "data", "file", "vbscript":
		return fmt.Errorf("redirect URI scheme is not allowed: %s", uri)
	default:
		// Private-use scheme for native apps; require a reverse-domain style name.
		if !strings.Contains(parsed.Scheme, ".") {
			return fmt.Errorf("custom redirect URI schemes must be reverse-domain names: %s", uri)
		}
	}
	return nil
}

// GetWendingOAuthTokenByKey reads a token straight from the database (no cache)
// for the low-traffic userinfo endpoint.
func GetWendingOAuthTokenByKey(key string) (*Token, error) {
	if key == "" {
		return nil, ErrTokenInvalid
	}
	var token Token
	if err := DB.Where(&Token{Key: key}).First(&token).Error; err != nil {
		return nil, err
	}
	return &token, nil
}
