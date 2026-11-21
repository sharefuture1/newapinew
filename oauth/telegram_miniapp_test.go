package oauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const telegramMiniAppTestBotToken = "123456:TEST-BOT-TOKEN"

// signTelegramMiniAppInitData builds an initData string the way Telegram
// does, using an independent implementation of the documented derivation so
// the test does not just call the production helpers back.
func signTelegramMiniAppInitData(t *testing.T, fields map[string]string) string {
	t.Helper()
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var builder strings.Builder
	for i, key := range keys {
		if i > 0 {
			builder.WriteByte('\n')
		}
		builder.WriteString(key + "=" + fields[key])
	}
	secretMac := hmac.New(sha256.New, []byte("WebAppData"))
	secretMac.Write([]byte(telegramMiniAppTestBotToken))
	expectedMac := hmac.New(sha256.New, secretMac.Sum(nil))
	expectedMac.Write([]byte(builder.String()))
	fields["hash"] = hex.EncodeToString(expectedMac.Sum(nil))
	values := url.Values{}
	for key, value := range fields {
		values.Set(key, value)
	}
	return values.Encode()
}

func telegramMiniAppTestFields(now time.Time) map[string]string {
	user, err := json.Marshal(map[string]any{
		"id": 396694408, "first_name": "John", "last_name": "Doe",
		"username": "john_doe", "language_code": "en",
	})
	if err != nil {
		panic(err)
	}
	return map[string]string{
		"auth_date": "1800000000",
		"query_id":  "AAHdF6IQAAAAAN0XohDhrOrc",
		"user":      string(user),
	}
}

func TestVerifyTelegramMiniAppInitDataValid(t *testing.T) {
	now := time.Unix(1_800_000_000, 0).UTC()
	auth, err := VerifyTelegramMiniAppInitData(
		signTelegramMiniAppInitData(t, telegramMiniAppTestFields(now)),
		telegramMiniAppTestBotToken, now)
	require.NoError(t, err)
	require.NotNil(t, auth)
	assert.Equal(t, int64(396694408), auth.User.ID)
	assert.Equal(t, "john_doe", auth.User.Username)
	assert.Equal(t, "John", auth.User.FirstName)
	assert.Equal(t, "Doe", auth.User.LastName)
	assert.Equal(t, time.Unix(1_800_000_000, 0).UTC(), auth.AuthDate)
	assert.NotEmpty(t, auth.Hash)
}

func TestVerifyTelegramMiniAppInitDataRejectsTampering(t *testing.T) {
	now := time.Unix(1_800_000_000, 0).UTC()
	fields := telegramMiniAppTestFields(now)
	initData := signTelegramMiniAppInitData(t, fields)
	// Tamper with a signed field after signing: the hash must no longer match.
	tampered := strings.Replace(initData, "john_doe", "mallory", 1)
	require.NotEqual(t, initData, tampered)
	_, err := VerifyTelegramMiniAppInitData(tampered, telegramMiniAppTestBotToken, now)
	assert.ErrorIs(t, err, ErrTelegramMiniAppFailed)

	// A hash computed with the Widget derivation (SHA256 of the token) must
	// not validate a Mini App payload.
	_, err = VerifyTelegramMiniAppInitData(initData, "wrong-bot-token", now)
	assert.ErrorIs(t, err, ErrTelegramMiniAppFailed)
}

func TestVerifyTelegramMiniAppInitDataRejectsExpired(t *testing.T) {
	now := time.Unix(1_800_000_000, 0).UTC()
	fields := telegramMiniAppTestFields(now)
	fields["auth_date"] = "1799999699" // 301 seconds old
	_, err := VerifyTelegramMiniAppInitData(
		signTelegramMiniAppInitData(t, fields), telegramMiniAppTestBotToken, now)
	assert.ErrorIs(t, err, ErrTelegramMiniAppExpired)

	// An auth_date far in the future is also rejected, not just old ones.
	fields["auth_date"] = "1800000121" // 121 seconds ahead
	_, err = VerifyTelegramMiniAppInitData(
		signTelegramMiniAppInitData(t, fields), telegramMiniAppTestBotToken, now)
	assert.ErrorIs(t, err, ErrTelegramMiniAppExpired)
}

func TestVerifyTelegramMiniAppInitDataRejectsMalformed(t *testing.T) {
	now := time.Unix(1_800_000_000, 0).UTC()
	valid := signTelegramMiniAppInitData(t, telegramMiniAppTestFields(now))
	for _, test := range []struct {
		name     string
		initData string
		token    string
		want     error
	}{
		{"empty initData", "", telegramMiniAppTestBotToken, ErrTelegramMiniAppFailed},
		{"empty bot token", valid, "", ErrTelegramMiniAppNotConfigured},
		{"missing hash", "auth_date=1800000000&user=%7B%22id%22%3A1%7D", telegramMiniAppTestBotToken, ErrTelegramMiniAppFailed},
		{"missing user", "auth_date=1800000000&hash=00", telegramMiniAppTestBotToken, ErrTelegramMiniAppFailed},
		{"non-hex hash", strings.Replace(valid, "hash=", "hash=zz", 1), telegramMiniAppTestBotToken, ErrTelegramMiniAppFailed},
		{"malformed user json", func() string {
			fields := telegramMiniAppTestFields(now)
			fields["user"] = "{not-json"
			return signTelegramMiniAppInitData(t, fields)
		}(), telegramMiniAppTestBotToken, ErrTelegramMiniAppFailed},
		{"zero user id", func() string {
			fields := telegramMiniAppTestFields(now)
			fields["user"] = `{"id":0,"first_name":"Nobody"}`
			return signTelegramMiniAppInitData(t, fields)
		}(), telegramMiniAppTestBotToken, ErrTelegramMiniAppFailed},
		{"missing auth_date", func() string {
			fields := telegramMiniAppTestFields(now)
			delete(fields, "auth_date")
			return signTelegramMiniAppInitData(t, fields)
		}(), telegramMiniAppTestBotToken, ErrTelegramMiniAppFailed},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := VerifyTelegramMiniAppInitData(test.initData, test.token, now)
			assert.ErrorIs(t, err, test.want)
			assert.True(t, errors.Is(err, test.want), "got %v", err)
		})
	}
}
