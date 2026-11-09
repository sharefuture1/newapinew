package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

const wendingShopRedirect = "https://shopapi.example.com/api/v1/auth/wendingai/callback"

func setupWendingLoginTest(t *testing.T, trusted bool) *wendingOAuthTestEnv {
	env := setupWendingOAuthTest(t, false)
	require.NoError(t, model.DB.Model(&model.User{}).Where("id = ?", env.userId).Update("email", "alice@example.com").Error)
	env.client.Trusted = trusted
	require.NoError(t, env.client.SetRedirectURIs([]string{wendingTestRedirect, wendingShopRedirect}))
	require.NoError(t, env.client.Save())
	return env
}

func (env *wendingOAuthTestEnv) approveScope(challenge, redirect, scope string) (string, map[string]any) {
	payload := map[string]any{
		"response_type": "code", "client_id": env.client.ClientId, "redirect_uri": redirect,
		"state": "st", "scope": scope, "code_challenge": challenge, "code_challenge_method": "S256",
		"approve": true,
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

func (env *wendingOAuthTestEnv) userinfo(bearer string) (*httptest.ResponseRecorder, map[string]any) {
	req := httptest.NewRequest(http.MethodGet, "/api/oauth/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+bearer)
	return env.do(req)
}

func TestWendingLoginScopeIssuesOpaqueTokenWithoutAPIKey(t *testing.T) {
	env := setupWendingLoginTest(t, false)
	verifier, challenge := wendingPKCE()
	code, body := env.approveScope(challenge, wendingShopRedirect, "login email")
	require.NotEmpty(t, code, "approve failed: %v", body)

	rec, resp := env.exchange(env.tokenForm(code, verifier, wendingShopRedirect))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
	access := resp["access_token"].(string)
	require.True(t, strings.HasPrefix(access, model.WendingOAuthAccessTokenPrefix))
	require.EqualValues(t, model.WendingOAuthAccessTokenTTLSeconds, resp["expires_in"])
	require.Equal(t, "login email", resp["scope"])

	// No API key was created for a sign-in grant.
	var tokens int64
	model.DB.Model(&model.Token{}).Where("user_id = ?", env.userId).Count(&tokens)
	require.EqualValues(t, 0, tokens)
	// Only the hash is stored.
	var stored model.WendingOAuthAccessToken
	require.NoError(t, model.DB.First(&stored).Error)
	require.NotContains(t, stored.TokenHash, access)
	require.Equal(t, model.WendingOAuthHash(access), stored.TokenHash)

	rec, info := env.userinfo(access)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Equal(t, fmt.Sprint(env.userId), info["sub"])
	require.Equal(t, "alice", info["username"])
	require.Equal(t, "Alice", info["display_name"])
	require.Equal(t, "alice@example.com", info["email"])
	require.Equal(t, false, info["email_verified"])
	require.Nil(t, info["user_quota"], "sign-in tokens must not expose balances")

	// code is single-use
	rec, resp = env.exchange(env.tokenForm(code, verifier, wendingShopRedirect))
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, "invalid_grant", resp["error"])
}

func TestWendingLoginTokenExpiresAndDiesWithGrant(t *testing.T) {
	env := setupWendingLoginTest(t, false)
	verifier, challenge := wendingPKCE()
	code, _ := env.approveScope(challenge, wendingShopRedirect, "login")
	_, resp := env.exchange(env.tokenForm(code, verifier, wendingShopRedirect))
	access := resp["access_token"].(string)

	// expired
	model.DB.Model(&model.WendingOAuthAccessToken{}).Where("1 = 1").Update("expires_at", common.GetTimestamp()-1)
	rec, _ := env.userinfo(access)
	require.Equal(t, http.StatusUnauthorized, rec.Code)

	// fresh token, then revoke the grant
	code, _ = env.approveScope(challenge, wendingShopRedirect, "login")
	_, resp = env.exchange(env.tokenForm(code, verifier, wendingShopRedirect))
	access = resp["access_token"].(string)
	rec, _ = env.userinfo(access)
	require.Equal(t, http.StatusOK, rec.Code)
	grant, err := model.GetActiveWendingOAuthGrant(env.userId, env.client.ClientId)
	require.NoError(t, err)
	require.Equal(t, 0, grant.TokenId)
	rec, body := env.do(httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/wending/oauth/grants/%d", grant.Id), nil))
	require.Equal(t, true, body["success"], rec.Body.String())
	rec, _ = env.userinfo(access)
	require.Equal(t, http.StatusUnauthorized, rec.Code)

	// garbage / wrong-prefix tokens are rejected
	rec, _ = env.userinfo(model.WendingOAuthAccessTokenPrefix + "nope")
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestWendingLoginScopeValidation(t *testing.T) {
	env := setupWendingLoginTest(t, true)
	_, challenge := wendingPKCE()
	for _, bad := range []string{"login api", "email", "api email", "openid"} {
		code, body := env.approveScope(challenge, wendingShopRedirect, bad)
		require.Empty(t, code, bad)
		require.Equal(t, "invalid_scope", body["oauth_error"], bad)
	}
}

func TestWendingLoginSkipConsentOnlyForTrustedLogin(t *testing.T) {
	for _, tc := range []struct {
		trusted bool
		scope   string
		skip    bool
		login   bool
	}{
		{true, "login", true, true},
		{true, "login email", true, true},
		{true, "api", false, false},
		{false, "login", false, true},
	} {
		t.Run(fmt.Sprintf("%v_%s", tc.trusted, strings.ReplaceAll(tc.scope, " ", "_")), func(t *testing.T) {
			env := setupWendingLoginTest(t, tc.trusted)
			_, challenge := wendingPKCE()
			query := url.Values{"response_type": {"code"}, "client_id": {env.client.ClientId}, "redirect_uri": {wendingShopRedirect},
				"state": {"s"}, "scope": {tc.scope}, "code_challenge": {challenge}, "code_challenge_method": {"S256"}}
			_, body := env.do(httptest.NewRequest(http.MethodGet, "/api/wending/oauth/authorize?"+query.Encode(), nil))
			require.Equal(t, true, body["success"], body)
			data := body["data"].(map[string]any)
			require.Equal(t, tc.skip, data["skip_consent"], tc)
			require.Equal(t, tc.login, data["login_only"], tc)
		})
	}
}

func TestWendingLoginKeepsExistingAPIKey(t *testing.T) {
	env := setupWendingLoginTest(t, true)
	verifier, challenge := wendingPKCE()
	// existing API grant with a key
	code, _ := env.approve(challenge, wendingTestRedirect)
	_, apiResp := env.exchange(env.tokenForm(code, verifier, wendingTestRedirect))
	key := apiResp["access_token"].(string)

	code, body := env.approveScope(challenge, wendingShopRedirect, "login")
	require.NotEmpty(t, code, "approve failed: %v", body)
	rec, resp := env.exchange(env.tokenForm(code, verifier, wendingShopRedirect))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.True(t, strings.HasPrefix(resp["access_token"].(string), model.WendingOAuthAccessTokenPrefix))

	// API key still valid and was not rotated
	_, err := model.GetWendingOAuthTokenByKey(strings.TrimPrefix(key, "sk-"))
	require.NoError(t, err)
	grant, err := model.GetActiveWendingOAuthGrant(env.userId, env.client.ClientId)
	require.NoError(t, err)
	require.True(t, model.WendingOAuthScopeHas(grant.Scope, "api"))
	require.True(t, model.WendingOAuthScopeHas(grant.Scope, "login"))
	// a sign-in token cannot be used as an sk- key
	rec, _ = env.userinfo("sk-" + resp["access_token"].(string))
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestWendingClientTrustedFlagPersists(t *testing.T) {
	env := setupWendingLoginTest(t, false)
	require.NoError(t, applyWendingOAuthClientRequest(env.client, wendingOAuthClientRequest{
		Name: "WendingAI Shop", RedirectUris: []string{wendingShopRedirect}, Enabled: true, Trusted: true,
	}))
	require.NoError(t, env.client.Save())
	reloaded, err := model.GetWendingOAuthClientById(env.client.Id)
	require.NoError(t, err)
	require.True(t, reloaded.Trusted)
}
