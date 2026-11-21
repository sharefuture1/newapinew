package controller

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const telegramMiniAppTestBotToken = "123456:TEST-BOT-TOKEN"

// signTelegramMiniAppInitData builds an initData string the way Telegram
// does, with an independent implementation of the documented derivation.
func signTelegramMiniAppInitData(t *testing.T, telegramID int64, authDate time.Time) string {
	t.Helper()
	user, err := json.Marshal(map[string]any{
		"id": telegramID, "first_name": "Mini", "last_name": "App",
		"username": "miniapp_user", "language_code": "en",
	})
	require.NoError(t, err)
	fields := map[string]string{
		"auth_date": strconv.FormatInt(authDate.Unix(), 10),
		"query_id":  "AAHdF6IQAAAAAN0XohDhrOrc",
		"user":      string(user),
	}
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

func telegramMiniAppLogin(t *testing.T, initData string) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(map[string]string{"init_data": initData})
	require.NoError(t, err)
	return securityEnrollmentRequest("POST", "/api/oauth/telegram/miniapp/login", string(body), "", service.AuthIdentity{}, TelegramMiniAppLogin)
}

func setupTelegramMiniAppTest(t *testing.T) *telegramOAuthFixture {
	t.Helper()
	fixture := setupTelegramOAuthTest(t)
	previousBotToken := common.TelegramBotToken
	common.TelegramBotToken = telegramMiniAppTestBotToken
	t.Cleanup(func() { common.TelegramBotToken = previousBotToken })
	return fixture
}

func TestTelegramMiniAppLoginExistingUser(t *testing.T) {
	fixture := setupTelegramMiniAppTest(t)
	const telegramID = 396694408
	require.NoError(t, model.DB.Model(fixture.user).Update("telegram_id", strconv.Itoa(telegramID)).Error)
	response := telegramMiniAppLogin(t, signTelegramMiniAppInitData(t, telegramID, time.Now()))
	var body securityEnrollmentResponse
	require.NoError(t, common.Unmarshal(response.Body.Bytes(), &body))
	require.True(t, body.Success, response.Body.String())
	var login struct {
		AccessToken string `json:"access_token"`
		User        struct {
			ID int `json:"id"`
		} `json:"user"`
	}
	require.NoError(t, common.Unmarshal(body.Data, &login))
	assert.NotEmpty(t, login.AccessToken)
	assert.Equal(t, fixture.user.Id, login.User.ID)
}

func TestTelegramMiniAppLoginAutoCreatesUser(t *testing.T) {
	fixture := setupTelegramMiniAppTest(t)
	const telegramID = 777001
	var before int64
	require.NoError(t, model.DB.Model(&model.User{}).Count(&before).Error)
	response := telegramMiniAppLogin(t, signTelegramMiniAppInitData(t, telegramID, time.Now()))
	var body securityEnrollmentResponse
	require.NoError(t, common.Unmarshal(response.Body.Bytes(), &body))
	require.True(t, body.Success, response.Body.String())
	var login struct {
		AccessToken string `json:"access_token"`
		User        struct {
			ID       int    `json:"id"`
			Username string `json:"username"`
		} `json:"user"`
	}
	require.NoError(t, common.Unmarshal(body.Data, &login))
	assert.NotEmpty(t, login.AccessToken)
	assert.NotEqual(t, fixture.user.Id, login.User.ID)
	// The free Telegram username is adopted, mirroring the OIDC creation path.
	assert.Equal(t, "miniapp_user", login.User.Username)
	stored, err := model.GetUserByTelegramID(strconv.Itoa(telegramID))
	require.NoError(t, err)
	assert.Equal(t, login.User.ID, stored.Id)
	assert.Equal(t, "miniapp_user", stored.Username)
	var after int64
	require.NoError(t, model.DB.Model(&model.User{}).Count(&after).Error)
	assert.Equal(t, before+1, after)
}

func TestTelegramMiniAppLoginFallsBackWhenUsernameTaken(t *testing.T) {
	setupTelegramMiniAppTest(t)
	owner := &model.User{Username: "miniapp_user", AffCode: "miniapp-owner", Status: common.UserStatusEnabled, AuthVersion: 1}
	require.NoError(t, model.DB.Create(owner).Error)
	const telegramID = 777002
	response := telegramMiniAppLogin(t, signTelegramMiniAppInitData(t, telegramID, time.Now()))
	var body securityEnrollmentResponse
	require.NoError(t, common.Unmarshal(response.Body.Bytes(), &body))
	require.True(t, body.Success, response.Body.String())
	var login struct {
		User struct {
			Username string `json:"username"`
		} `json:"user"`
	}
	require.NoError(t, common.Unmarshal(body.Data, &login))
	assert.Contains(t, login.User.Username, "telegram_")
	stored, err := model.GetUserByTelegramID(strconv.Itoa(telegramID))
	require.NoError(t, err)
	assert.Equal(t, login.User.Username, stored.Username)
}

func TestTelegramMiniAppLoginRejectsReplay(t *testing.T) {
	fixture := setupTelegramMiniAppTest(t)
	const telegramID = 396694408
	require.NoError(t, model.DB.Model(fixture.user).Update("telegram_id", strconv.Itoa(telegramID)).Error)
	initData := signTelegramMiniAppInitData(t, telegramID, time.Now())
	first := telegramMiniAppLogin(t, initData)
	assert.Contains(t, first.Body.String(), `"success":true`)
	second := telegramMiniAppLogin(t, initData)
	assert.Contains(t, second.Body.String(), "TELEGRAM_MINIAPP_FAILED")
	assert.NotContains(t, second.Body.String(), "access_token")
}

func TestTelegramMiniAppLoginRejectsInvalidInitData(t *testing.T) {
	fixture := setupTelegramMiniAppTest(t)
	const telegramID = 396694408
	require.NoError(t, model.DB.Model(fixture.user).Update("telegram_id", strconv.Itoa(telegramID)).Error)
	now := time.Now()
	valid := signTelegramMiniAppInitData(t, telegramID, now)
	tampered := strings.Replace(valid, "miniapp_user", "mallory", 1)
	require.NotEqual(t, valid, tampered)
	for _, test := range []struct {
		name     string
		initData string
		want     string
	}{
		{"tampered user", tampered, "TELEGRAM_MINIAPP_FAILED"},
		{"expired", signTelegramMiniAppInitData(t, telegramID, now.Add(-10*time.Minute)), "TELEGRAM_MINIAPP_EXPIRED"},
		{"empty", "", `"success":false`},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := telegramMiniAppLogin(t, test.initData)
			assert.Contains(t, response.Body.String(), test.want)
			assert.NotContains(t, response.Body.String(), "access_token")
		})
	}
}

func TestTelegramMiniAppLoginRequiresBotToken(t *testing.T) {
	setupTelegramMiniAppTest(t)
	common.TelegramBotToken = ""
	response := telegramMiniAppLogin(t, signTelegramMiniAppInitData(t, 396694408, time.Now()))
	assert.Contains(t, response.Body.String(), "TELEGRAM_MINIAPP_NOT_CONFIGURED")
	assert.NotContains(t, response.Body.String(), "access_token")
}
