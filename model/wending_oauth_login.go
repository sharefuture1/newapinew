package model

// Wending OAuth "login" scope: sign in to a first-party product with a
// WendingAI account WITHOUT creating an API key. The authorization code is
// exchanged for a short-lived opaque access token (prefix wdat_, only its
// SHA-256 is stored) that is good for /api/oauth/userinfo only.
//
// Fork-local customization; see WENDING_PATCHES.md and docs/WENDING_OAUTH.md.

import (
	"errors"
	"slices"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	WendingOAuthScopeAPI     = "api"
	WendingOAuthScopeProfile = "profile"
	WendingOAuthScopeLogin   = "login"
	WendingOAuthScopeEmail   = "email"

	WendingOAuthAccessTokenPrefix     = "wdat_"
	WendingOAuthAccessTokenTTLSeconds = 600 // 10 minutes: enough for the RP to call userinfo once
)

var ErrWendingOAuthAccessTokenInvalid = errors.New("access token is invalid or expired")

// WendingOAuthAccessToken is a short-lived sign-in token (login scope).
type WendingOAuthAccessToken struct {
	Id        int    `json:"id"`
	TokenHash string `json:"-" gorm:"type:varchar(64);uniqueIndex"`
	ClientId  string `json:"client_id" gorm:"type:varchar(64);index"`
	UserId    int    `json:"user_id" gorm:"index"`
	GrantId   int    `json:"grant_id" gorm:"index"`
	Scope     string `json:"scope" gorm:"type:varchar(255)"`
	ExpiresAt int64  `json:"expires_at" gorm:"bigint;index"`
	CreatedAt int64  `json:"created_at" gorm:"bigint"`
}

func (WendingOAuthAccessToken) TableName() string { return "wending_oauth_access_tokens" }

// WendingOAuthScopeHas reports whether the space separated scope contains item.
func WendingOAuthScopeHas(scope, item string) bool {
	return slices.Contains(strings.Fields(scope), item)
}

// WendingOAuthIsLoginScope reports whether scope is a sign-in-only request:
// it contains "login" and never "api" (no API key is issued).
func WendingOAuthIsLoginScope(scope string) bool {
	return WendingOAuthScopeHas(scope, WendingOAuthScopeLogin) && !WendingOAuthScopeHas(scope, WendingOAuthScopeAPI)
}

// WendingOAuthMergeScopes returns the union of two scope strings (stable order).
func WendingOAuthMergeScopes(a, b string) string {
	out := make([]string, 0, 4)
	for _, item := range append(strings.Fields(a), strings.Fields(b)...) {
		if !slices.Contains(out, item) {
			out = append(out, item)
		}
	}
	return strings.Join(out, " ")
}

// IssueWendingOAuthLoginGrant records (or refreshes) the user's sign-in grant
// for client. It never creates or rotates an API key: an existing API grant
// for the same client keeps its key and simply gains the login scope.
func IssueWendingOAuthLoginGrant(userId int, client *WendingOAuthClient, scope string) (*WendingOAuthGrant, bool, error) {
	existing, err := GetActiveWendingOAuthGrant(userId, client.ClientId)
	if err != nil {
		return nil, false, err
	}
	now := common.GetTimestamp()
	if existing != nil {
		merged := WendingOAuthMergeScopes(existing.Scope, scope)
		if err := DB.Model(&WendingOAuthGrant{}).Where("id = ?", existing.Id).
			Updates(map[string]any{"scope": merged, "updated_at": now}).Error; err != nil {
			return nil, false, err
		}
		existing.Scope, existing.UpdatedAt = merged, now
		return existing, true, nil
	}
	grant := &WendingOAuthGrant{UserId: userId, ClientId: client.ClientId, Scope: scope, CreatedAt: now, UpdatedAt: now}
	if err := DB.Create(grant).Error; err != nil {
		return nil, false, err
	}
	return grant, false, nil
}

// CreateWendingOAuthAccessToken issues a short-lived sign-in token and returns its plaintext.
func CreateWendingOAuthAccessToken(grant *WendingOAuthGrant, scope string) (string, int64, error) {
	plain, err := WendingOAuthRandomToken(WendingOAuthAccessTokenPrefix)
	if err != nil {
		return "", 0, err
	}
	now := common.GetTimestamp()
	row := &WendingOAuthAccessToken{
		TokenHash: WendingOAuthHash(plain),
		ClientId:  grant.ClientId,
		UserId:    grant.UserId,
		GrantId:   grant.Id,
		Scope:     scope,
		ExpiresAt: now + WendingOAuthAccessTokenTTLSeconds,
		CreatedAt: now,
	}
	if err := DB.Create(row).Error; err != nil {
		return "", 0, err
	}
	// Opportunistic cleanup; failures are harmless.
	DB.Where("expires_at < ?", now-86400).Delete(&WendingOAuthAccessToken{})
	return plain, WendingOAuthAccessTokenTTLSeconds, nil
}

// ValidateWendingOAuthAccessToken returns the token row and its active grant.
func ValidateWendingOAuthAccessToken(plain string) (*WendingOAuthAccessToken, *WendingOAuthGrant, error) {
	if !strings.HasPrefix(plain, WendingOAuthAccessTokenPrefix) {
		return nil, nil, ErrWendingOAuthAccessTokenInvalid
	}
	var row WendingOAuthAccessToken
	if err := DB.Where("token_hash = ?", WendingOAuthHash(plain)).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrWendingOAuthAccessTokenInvalid
		}
		return nil, nil, err
	}
	if row.ExpiresAt < common.GetTimestamp() {
		return nil, nil, ErrWendingOAuthAccessTokenInvalid
	}
	grant, err := GetWendingOAuthGrantById(row.GrantId)
	if err != nil || grant.RevokedAt != 0 || grant.UserId != row.UserId || grant.ClientId != row.ClientId {
		return nil, nil, ErrWendingOAuthAccessTokenInvalid
	}
	return &row, grant, nil
}
