package controller

// Tests for native (desktop) client support: RFC 8252 loopback redirects,
// per-device grants and the public client-info endpoint.

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

const wendingLoopbackRegistered = "http://127.0.0.1/wending/callback"

func (env *wendingOAuthTestEnv) approveDevice(challenge, redirect, deviceId, deviceName string) (string, map[string]any) {
	payload := map[string]any{
		"response_type": "code", "client_id": env.client.ClientId, "redirect_uri": redirect,
		"state": "xyz", "code_challenge": challenge, "code_challenge_method": "S256",
		"approve": true, "remain_quota": 250000, "device_id": deviceId, "device_name": deviceName,
	}
	data, _ := common.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/wending/oauth/authorize", strings.NewReader(string(data)))
	req.Header.Set("Content-Type", "application/json")
	_, body := env.do(req)
	if body["success"] != true {
		return "", body
	}
	target, err := url.Parse(body["data"].(map[string]any)["redirect_to"].(string))
	require.NoError(env.t, err)
	return target.Query().Get("code"), body
}

func setupDesktopClient(t *testing.T, isPublic bool) *wendingOAuthTestEnv {
	env := setupWendingOAuthTest(t, isPublic)
	require.NoError(t, env.client.SetRedirectURIs([]string{wendingTestRedirect, wendingLoopbackRegistered}))
	require.NoError(t, model.DB.Model(env.client).Update("redirect_uris", env.client.RedirectUris).Error)
	env.router.GET("/api/oauth/clients/:client_id", GetWendingOAuthPublicClient)
	return env
}

func TestWendingLoopbackRedirectAnyPortForPublicClient(t *testing.T) {
	env := setupDesktopClient(t, true)
	verifier, challenge := wendingPKCE()
	redirect := "http://127.0.0.1:53127/wending/callback"
	code, body := env.approve(challenge, redirect)
	require.NotEmpty(t, code, body)
	form := env.tokenForm(code, verifier, redirect)
	form.Del("client_secret")
	rec, resp := env.exchange(form)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.True(t, strings.HasPrefix(resp["access_token"].(string), "sk-"))

	// The token request must use the same redirect_uri (including the port).
	code, _ = env.approve(challenge, redirect)
	form = env.tokenForm(code, verifier, "http://127.0.0.1:53128/wending/callback")
	form.Del("client_secret")
	rec, resp = env.exchange(form)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, "invalid_grant", resp["error"])
}

func TestWendingLoopbackRedirectRules(t *testing.T) {
	public := &model.WendingOAuthClient{IsPublic: true}
	require.NoError(t, public.SetRedirectURIs([]string{wendingLoopbackRegistered, "http://[::1]/cb"}))
	for _, ok := range []string{
		"http://127.0.0.1/wending/callback",
		"http://127.0.0.1:1/wending/callback",
		"http://127.0.0.1:65535/wending/callback",
		"http://[::1]:4000/cb",
	} {
		require.True(t, public.MatchRedirectURI(ok), ok)
	}
	for _, bad := range []string{
		"http://127.0.0.1:53127/wending/callback/x",
		"http://127.0.0.1:53127/other",
		"http://localhost:53127/wending/callback",
		"https://127.0.0.1:53127/wending/callback",
		"http://127.0.0.2:53127/wending/callback",
		"http://127.0.0.1:53127/wending/callback?x=1",
		"http://user@127.0.0.1:53127/wending/callback",
		"http://127.0.0.1:99999/wending/callback",
		"",
	} {
		require.False(t, public.MatchRedirectURI(bad), bad)
	}
	confidential := &model.WendingOAuthClient{IsPublic: false}
	require.NoError(t, confidential.SetRedirectURIs([]string{wendingLoopbackRegistered}))
	require.True(t, confidential.MatchRedirectURI(wendingLoopbackRegistered))
	require.False(t, confidential.MatchRedirectURI("http://127.0.0.1:53127/wending/callback"),
		"confidential clients keep exact matching")
}

func TestWendingDeviceGrantsAreIndependent(t *testing.T) {
	env := setupDesktopClient(t, true)
	verifier, challenge := wendingPKCE()
	redirect := "http://127.0.0.1:40001/wending/callback"
	exchange := func(code string) map[string]any {
		form := env.tokenForm(code, verifier, redirect)
		form.Del("client_secret")
		rec, resp := env.exchange(form)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		return resp
	}

	codeA, _ := env.approveDevice(challenge, redirect, "device-aaaaaaaa", "MacBook\x07 <Pro>")
	keyA := exchange(codeA)["access_token"].(string)
	codeB, _ := env.approveDevice(challenge, redirect, "device-bbbbbbbb", "Windows 台式机")
	keyB := exchange(codeB)["access_token"].(string)
	require.NotEqual(t, keyA, keyB)

	// Authorizing device B did not break device A.
	_, err := model.GetWendingOAuthTokenByKey(strings.TrimPrefix(keyA, "sk-"))
	require.NoError(t, err, "device A key must keep working")

	// Re-authorizing device A rotates only A's key.
	codeA2, _ := env.approveDevice(challenge, redirect, "device-aaaaaaaa", "MacBook Pro")
	keyA2 := exchange(codeA2)["access_token"].(string)
	require.NotEqual(t, keyA, keyA2)
	_, err = model.GetWendingOAuthTokenByKey(strings.TrimPrefix(keyA, "sk-"))
	require.Error(t, err, "old key of device A must stop working")
	_, err = model.GetWendingOAuthTokenByKey(strings.TrimPrefix(keyB, "sk-"))
	require.NoError(t, err, "device B key must keep working")

	var grants []model.WendingOAuthGrant
	require.NoError(t, model.DB.Where("user_id = ? AND revoked_at = 0", env.userId).Order("id").Find(&grants).Error)
	require.Len(t, grants, 2)
	require.Equal(t, "MacBook Pro", grants[0].DeviceName)
	require.Equal(t, "Windows 台式机", grants[1].DeviceName)

	var tokens []model.Token
	require.NoError(t, model.DB.Where("user_id = ?", env.userId).Order("id").Find(&tokens).Error)
	require.Len(t, tokens, 2)
	require.Equal(t, "授权:WendingAI 创作台 · MacBook Pro", tokens[0].Name)
	require.LessOrEqual(t, len(tokens[0].Name), 50)

	// Web clients (no device id) still get their own single grant.
	codeWeb, _ := env.approve(challenge, redirect)
	exchange(codeWeb)
	var count int64
	model.DB.Model(&model.WendingOAuthGrant{}).Where("user_id = ? AND revoked_at = 0", env.userId).Count(&count)
	require.EqualValues(t, 3, count)
}

func TestWendingDeviceIdValidation(t *testing.T) {
	env := setupDesktopClient(t, true)
	_, challenge := wendingPKCE()
	_, body := env.approveDevice(challenge, "http://127.0.0.1:40001/wending/callback", "bad id!", "x")
	require.NotEqual(t, true, body["success"])
	require.Equal(t, "invalid_request", body["oauth_error"])

	// The GET (consent screen) reports the cleaned device name.
	q := url.Values{"response_type": {"code"}, "client_id": {env.client.ClientId},
		"redirect_uri": {"http://127.0.0.1:40001/wending/callback"}, "state": {"s"},
		"code_challenge": {challenge}, "code_challenge_method": {"S256"},
		"device_id": {"device-cccccccc"}, "device_name": {strings.Repeat("长", 40)}}
	req := httptest.NewRequest(http.MethodGet, "/api/wending/oauth/authorize?"+q.Encode(), nil)
	_, resp := env.do(req)
	require.Equal(t, true, resp["success"], resp)
	require.Equal(t, strings.Repeat("长", 32), resp["data"].(map[string]any)["device_name"])
}

func TestWendingPublicClientInfo(t *testing.T) {
	env := setupDesktopClient(t, true)
	req := httptest.NewRequest(http.MethodGet, "/api/oauth/clients/"+env.client.ClientId, nil)
	rec, body := env.do(req)
	require.Equal(t, http.StatusOK, rec.Code)
	data := body["data"].(map[string]any)
	require.Equal(t, env.client.ClientId, data["client_id"])
	require.Equal(t, true, data["is_public"])
	require.NotContains(t, rec.Body.String(), "secret")
	require.NotContains(t, rec.Body.String(), "redirect")

	rec, _ = env.do(httptest.NewRequest(http.MethodGet, "/api/oauth/clients/wd_unknown", nil))
	require.Equal(t, http.StatusNotFound, rec.Code)

	require.NoError(t, model.DB.Model(env.client).Update("enabled", false).Error)
	rec, _ = env.do(httptest.NewRequest(http.MethodGet, "/api/oauth/clients/"+env.client.ClientId, nil))
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestWendingCreateClientWithFixedId(t *testing.T) {
	env := setupWendingOAuthTest(t, true)
	env.router.POST("/api/wending/oauth/clients", CreateWendingOAuthClient)
	post := func(clientId string) (*httptest.ResponseRecorder, map[string]any) {
		payload := map[string]any{"client_id": clientId, "name": "WendingAI Switch", "is_public": true, "enabled": true,
			"redirect_uris": []string{wendingLoopbackRegistered}, "default_remain_quota": 500000}
		data, _ := common.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/api/wending/oauth/clients", strings.NewReader(string(data)))
		req.Header.Set("Content-Type", "application/json")
		return env.do(req)
	}
	rec, body := post("wd_wendingai_switch")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Equal(t, true, body["success"], body)
	client, err := model.GetWendingOAuthClientByClientId("wd_wendingai_switch")
	require.NoError(t, err)
	require.True(t, client.IsPublic)

	rec, body = post("wd_wendingai_switch")
	require.NotEqual(t, true, body["success"], "duplicate id must be rejected")
	for _, bad := range []string{"wendingai_switch", "wd_UPPER", "wd_a", "wd_bad-dash"} {
		_, body = post(bad)
		require.NotEqual(t, true, body["success"], bad)
	}
	// Empty id still generates a random one.
	rec, body = post("")
	require.Equal(t, true, body["success"], body)
	require.True(t, strings.HasPrefix(body["data"].(map[string]any)["client"].(map[string]any)["client_id"].(string), "wd_"))
}
