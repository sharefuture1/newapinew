package oauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Telegram Mini App login.
//
// This is separate from the Telegram OIDC login in telegram.go: OIDC goes
// through https://oauth.telegram.org with a ClientID/ClientSecret and refuses
// unbound users, while a Mini App opened from Telegram carries a signed
// initData payload that is verified with the bot token instead. The two
// credentials must not be mixed up: the OIDC client secret never validates
// initData, and the bot token never exchanges an OIDC code.
//
// Verification follows the Telegram WebApp validation contract, as already
// implemented by the 稳定点AI shop (dujiao-wending
// internal/modules/identity/telegramauth/application/service.go,
// VerifyMiniAppInitData):
//   - data_check_string is every initData field except "hash", sorted by key,
//     joined as "k=v" with "\n";
//   - secret = HMAC-SHA256(key="WebAppData", bot_token);
//   - expected hash = HMAC-SHA256(secret, data_check_string).
// This differs from the retired Login Widget derivation
// (secret = SHA256(bot_token)); do not reuse that one here.

const (
	// TelegramMiniAppAuthValidity is how long an initData payload is accepted
	// after its auth_date. It matches the shop's login_expire_seconds default.
	TelegramMiniAppAuthValidity = 300 * time.Second
	// TelegramMiniAppMaxClockSkew tolerates a client clock slightly ahead of
	// the server. An auth_date further in the future is rejected.
	TelegramMiniAppMaxClockSkew = 60 * time.Second
)

var (
	ErrTelegramMiniAppNotConfigured = errors.New("Telegram Mini App login is not configured. Please contact your administrator.")
	ErrTelegramMiniAppFailed        = errors.New("Telegram Mini App authorization failed. Please try again.")
	ErrTelegramMiniAppExpired       = errors.New("Telegram Mini App authorization has expired. Please reopen the Mini App and try again.")
)

// TelegramMiniAppUser is the Telegram user carried by a verified initData payload.
type TelegramMiniAppUser struct {
	ID        int64  `json:"id"`
	Username  string `json:"username,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	PhotoURL  string `json:"photo_url,omitempty"`
}

// TelegramMiniAppAuth is a verified Mini App authorization.
type TelegramMiniAppAuth struct {
	User     TelegramMiniAppUser
	AuthDate time.Time
	Hash     string
}

// VerifyTelegramMiniAppInitData verifies a Telegram WebApp initData string
// (the raw query-string payload from window.Telegram.WebApp.initData) with
// the bot token. The returned Hash is the caller-supplied hash, suitable as
// a one-time-use assertion for replay protection; it is never a secret.
func VerifyTelegramMiniAppInitData(initData, botToken string, now time.Time) (*TelegramMiniAppAuth, error) {
	if strings.TrimSpace(botToken) == "" {
		return nil, ErrTelegramMiniAppNotConfigured
	}
	if strings.TrimSpace(initData) == "" {
		return nil, ErrTelegramMiniAppFailed
	}
	values, err := url.ParseQuery(initData)
	if err != nil {
		return nil, ErrTelegramMiniAppFailed
	}
	hash := strings.TrimSpace(values.Get("hash"))
	if hash == "" {
		return nil, ErrTelegramMiniAppFailed
	}
	userRaw := values.Get("user")
	if userRaw == "" {
		return nil, ErrTelegramMiniAppFailed
	}
	var user TelegramMiniAppUser
	if err := json.Unmarshal([]byte(userRaw), &user); err != nil || user.ID <= 0 {
		return nil, ErrTelegramMiniAppFailed
	}
	authDateUnix, err := strconv.ParseInt(strings.TrimSpace(values.Get("auth_date")), 10, 64)
	if err != nil || authDateUnix <= 0 {
		return nil, ErrTelegramMiniAppFailed
	}
	authDate := time.Unix(authDateUnix, 0).UTC()
	if authDate.After(now.Add(TelegramMiniAppMaxClockSkew)) || now.Sub(authDate) > TelegramMiniAppAuthValidity {
		return nil, ErrTelegramMiniAppExpired
	}
	if !verifyTelegramMiniAppHash(values, hash, botToken) {
		return nil, ErrTelegramMiniAppFailed
	}
	return &TelegramMiniAppAuth{User: user, AuthDate: authDate, Hash: hash}, nil
}

func verifyTelegramMiniAppHash(values url.Values, hash, botToken string) bool {
	keys := make([]string, 0, len(values))
	for key := range values {
		if key == "hash" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var builder strings.Builder
	for i, key := range keys {
		if i > 0 {
			builder.WriteByte('\n')
		}
		fmt.Fprintf(&builder, "%s=%s", key, values.Get(key))
	}
	secret := telegramMiniAppHMAC([]byte("WebAppData"), []byte(botToken))
	expected := telegramMiniAppHMAC(secret, []byte(builder.String()))
	decoded, err := hex.DecodeString(hash)
	if err != nil {
		return false
	}
	return hmac.Equal(decoded, expected)
}

func telegramMiniAppHMAC(key, message []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(message)
	return mac.Sum(nil)
}
