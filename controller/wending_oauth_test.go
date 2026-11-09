package controller

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const wendingTestRedirect = "https://huabu.example.com/api/auth/wending/callback"

type wendingOAuthTestEnv struct {
	t      *testing.T
	router *gin.Engine
	userId int
	client *model.WendingOAuthClient
	secret string
}

func setupWendingOAuthTest(t *testing.T, isPublic bool) *wendingOAuthTestEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	common.RedisEnabled = false
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB, model.LOG_DB = db, db
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Token{}, &model.AuditLog{},
		&model.WendingOAuthClient{}, &model.WendingOAuthGrant{}, &model.WendingOAuthCode{}, &model.WendingOAuthAccessToken{}))

	user := &model.User{Username: "alice", Password: "x", DisplayName: "Alice", Role: common.RoleCommonUser,
		Status: common.UserStatusEnabled, Group: "default", Quota: 1000000, AffCode: "aff-alice", AuthVersion: 1}
	require.NoError(t, db.Create(user).Error)

	client := &model.WendingOAuthClient{Name: "WendingAI 创作台", IsPublic: isPublic, Enabled: true,
		DefaultRemainQuota: 500000, AllowUnlimitedQuota: true, ClientId: "wd_testclient000000000000"}
	require.NoError(t, client.SetRedirectURIs([]string{wendingTestRedirect}))
	secret, err := client.RotateSecret()
	require.NoError(t, err)
	require.NoError(t, client.Insert())

	router := gin.New()
	session := func(c *gin.Context) {
		c.Set("id", user.Id)
		c.Set("role", common.RoleCommonUser)
		c.Set("group", "default")
		c.Set("session_id", "sid-test")
		c.Set("auth_version", int64(1))
		c.Set("session_version", int64(1))
		c.Next()
	}
	router.GET("/api/wending/oauth/authorize", session, GetWendingOAuthAuthorize)
	router.POST("/api/wending/oauth/authorize", session, PostWendingOAuthAuthorize)
	router.GET("/api/wending/oauth/sidebar-apps", session, ListWendingOAuthSidebarApps)
	router.DELETE("/api/wending/oauth/grants/:id", session, RevokeSelfWendingOAuthGrant)
	router.POST("/api/oauth/token", PostWendingOAuthToken)
	router.GET("/api/oauth/userinfo", GetWendingOAuthUserInfo)
	return &wendingOAuthTestEnv{t: t, router: router, userId: user.Id, client: client, secret: secret}
}

func wendingPKCE() (string, string) {
	verifier := strings.Repeat("v", 20) + "-verifier._~0123456789abcdef"
	sum := sha256.Sum256([]byte(verifier))
	return verifier, base64.RawURLEncoding.EncodeToString(sum[:])
}

func (env *wendingOAuthTestEnv) do(req *http.Request) (*httptest.ResponseRecorder, map[string]any) {
	rec := httptest.NewRecorder()
	env.router.ServeHTTP(rec, req)
	var body map[string]any
	_ = common.Unmarshal(rec.Body.Bytes(), &body)
	return rec, body
}

// approve runs the consent POST and returns the code from the redirect.
func (env *wendingOAuthTestEnv) approve(challenge string, redirect string) (string, map[string]any) {
	payload := map[string]any{
		"response_type": "code", "client_id": env.client.ClientId, "redirect_uri": redirect,
		"state": "xyz", "code_challenge": challenge, "code_challenge_method": "S256",
		"approve": true, "remain_quota": 250000,
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
	require.Equal(env.t, "xyz", target.Query().Get("state"))
	return target.Query().Get("code"), body
}

func (env *wendingOAuthTestEnv) exchange(form url.Values) (*httptest.ResponseRecorder, map[string]any) {
	req := httptest.NewRequest(http.MethodPost, "/api/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return env.do(req)
}

func (env *wendingOAuthTestEnv) tokenForm(code, verifier, redirect string) url.Values {
	return url.Values{"grant_type": {"authorization_code"}, "code": {code}, "code_verifier": {verifier},
		"redirect_uri": {redirect}, "client_id": {env.client.ClientId}, "client_secret": {env.secret}}
}

func TestWendingOAuthSuccessAndCodeReuse(t *testing.T) {
	env := setupWendingOAuthTest(t, false)
	verifier, challenge := wendingPKCE()
	code, body := env.approve(challenge, wendingTestRedirect)
	require.NotEmpty(t, code, "approve failed: %v", body)

	rec, resp := env.exchange(env.tokenForm(code, verifier, wendingTestRedirect))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
	key := resp["access_token"].(string)
	require.True(t, strings.HasPrefix(key, "sk-"))
	require.Equal(t, "Bearer", resp["token_type"])

	token, err := model.GetWendingOAuthTokenByKey(strings.TrimPrefix(key, "sk-"))
	require.NoError(t, err)
	require.Equal(t, env.userId, token.UserId)
	require.Equal(t, 250000, token.RemainQuota)
	require.False(t, token.UnlimitedQuota)
	require.True(t, strings.HasPrefix(token.Name, model.WendingOAuthTokenNamePrefix))

	// userinfo with the issued key
	req := httptest.NewRequest(http.MethodGet, "/api/oauth/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+key)
	rec, info := env.do(req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Equal(t, "alice", info["username"])

	// code reuse is rejected
	rec, resp = env.exchange(env.tokenForm(code, verifier, wendingTestRedirect))
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, "invalid_grant", resp["error"])
}

func TestWendingOAuthPKCEFailure(t *testing.T) {
	env := setupWendingOAuthTest(t, false)
	_, challenge := wendingPKCE()
	code, body := env.approve(challenge, wendingTestRedirect)
	require.NotEmpty(t, code, "approve failed: %v", body)

	wrong := strings.Repeat("w", 50)
	rec, resp := env.exchange(env.tokenForm(code, wrong, wendingTestRedirect))
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, "invalid_grant", resp["error"])

	rec, resp = env.exchange(env.tokenForm(code, "", wendingTestRedirect))
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, "invalid_request", resp["error"])
}

func TestWendingOAuthAuthorizeRequiresS256(t *testing.T) {
	env := setupWendingOAuthTest(t, false)
	query := url.Values{"response_type": {"code"}, "client_id": {env.client.ClientId}, "redirect_uri": {wendingTestRedirect},
		"state": {"s"}, "code_challenge": {"plain-challenge"}, "code_challenge_method": {"plain"}}
	_, body := env.do(httptest.NewRequest(http.MethodGet, "/api/wending/oauth/authorize?"+query.Encode(), nil))
	require.Equal(t, false, body["success"])
	require.Equal(t, "invalid_request", body["oauth_error"])
	require.Equal(t, true, body["redirect_trusted"])
}

func TestWendingOAuthRedirectMismatch(t *testing.T) {
	env := setupWendingOAuthTest(t, false)
	verifier, challenge := wendingPKCE()

	// Unregistered redirect at authorize: rejected and NOT redirected.
	for _, bad := range []string{"https://evil.example.com/cb", wendingTestRedirect + "/", wendingTestRedirect + "?x=1"} {
		code, body := env.approve(challenge, bad)
		require.Empty(t, code)
		require.Equal(t, false, body["success"])
		require.Equal(t, false, body["redirect_trusted"])
		require.Nil(t, body["redirect_to"])
	}

	// Redirect at token endpoint must equal the one used at authorize.
	code, body := env.approve(challenge, wendingTestRedirect)
	require.NotEmpty(t, code, "approve failed: %v", body)
	rec, resp := env.exchange(env.tokenForm(code, verifier, "https://huabu.example.com/other"))
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, "invalid_grant", resp["error"])
}

func TestWendingOAuthClientAuthentication(t *testing.T) {
	env := setupWendingOAuthTest(t, false)
	verifier, challenge := wendingPKCE()
	code, _ := env.approve(challenge, wendingTestRedirect)

	form := env.tokenForm(code, verifier, wendingTestRedirect)
	form.Del("client_secret")
	rec, resp := env.exchange(form)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.Equal(t, "invalid_client", resp["error"])

	form.Set("client_secret", "wds_wrong")
	rec, _ = env.exchange(form)
	require.Equal(t, http.StatusUnauthorized, rec.Code)

	// client_secret_basic works
	form.Del("client_secret")
	req := httptest.NewRequest(http.MethodPost, "/api/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(env.client.ClientId, env.secret)
	rec, _ = env.do(req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}

func TestWendingOAuthPublicClientWithoutSecret(t *testing.T) {
	env := setupWendingOAuthTest(t, true)
	verifier, challenge := wendingPKCE()
	code, _ := env.approve(challenge, wendingTestRedirect)
	form := env.tokenForm(code, verifier, wendingTestRedirect)
	form.Del("client_secret")
	rec, _ := env.exchange(form)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}

func TestWendingOAuthRevokedGrant(t *testing.T) {
	env := setupWendingOAuthTest(t, false)
	verifier, challenge := wendingPKCE()
	code, _ := env.approve(challenge, wendingTestRedirect)
	rec, resp := env.exchange(env.tokenForm(code, verifier, wendingTestRedirect))
	require.Equal(t, http.StatusOK, rec.Code)
	key := resp["access_token"].(string)

	// a second, still-unused code for the same grant
	code2, _ := env.approve(challenge, wendingTestRedirect)
	grant, err := model.GetActiveWendingOAuthGrant(env.userId, env.client.ClientId)
	require.NoError(t, err)
	require.NotNil(t, grant)

	rec, body := env.do(httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/wending/oauth/grants/%d", grant.Id), nil))
	require.Equal(t, true, body["success"], rec.Body.String())

	// outstanding code is dead
	rec, resp = env.exchange(env.tokenForm(code2, verifier, wendingTestRedirect))
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, "invalid_grant", resp["error"])

	// key no longer validates and userinfo rejects it
	_, err = model.GetWendingOAuthTokenByKey(strings.TrimPrefix(key, "sk-"))
	require.Error(t, err)
	req := httptest.NewRequest(http.MethodGet, "/api/oauth/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+key)
	rec, _ = env.do(req)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestWendingOAuthReapproveRotatesKey(t *testing.T) {
	env := setupWendingOAuthTest(t, false)
	verifier, challenge := wendingPKCE()
	code, _ := env.approve(challenge, wendingTestRedirect)
	_, first := env.exchange(env.tokenForm(code, verifier, wendingTestRedirect))
	code, _ = env.approve(challenge, wendingTestRedirect)
	_, second := env.exchange(env.tokenForm(code, verifier, wendingTestRedirect))
	require.NotEqual(t, first["access_token"], second["access_token"])

	_, err := model.GetWendingOAuthTokenByKey(strings.TrimPrefix(first["access_token"].(string), "sk-"))
	require.Error(t, err, "old key must stop working after rotation")
	var count int64
	model.DB.Model(&model.Token{}).Where("user_id = ?", env.userId).Count(&count)
	require.EqualValues(t, 1, count)
}

func TestWendingOAuthExpiredCode(t *testing.T) {
	env := setupWendingOAuthTest(t, false)
	verifier, challenge := wendingPKCE()
	code, _ := env.approve(challenge, wendingTestRedirect)
	model.DB.Model(&model.WendingOAuthCode{}).Where("1 = 1").Update("expires_at", common.GetTimestamp()-1)
	rec, resp := env.exchange(env.tokenForm(code, verifier, wendingTestRedirect))
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, "invalid_grant", resp["error"])
}

func TestWendingRedirectURIValidation(t *testing.T) {
	for _, ok := range []string{"https://huabu.wending.ai/cb", "http://localhost:3000/cb", "http://127.0.0.1:8080/x", "ai.wending.mcp:/cb"} {
		require.NoError(t, model.ValidateWendingRedirectURI(ok), ok)
	}
	for _, bad := range []string{"http://huabu.wending.ai/cb", "https://a.com/cb#frag", "javascript:alert(1)", "/relative", "myapp:/cb"} {
		require.Error(t, model.ValidateWendingRedirectURI(bad), bad)
	}
}

func TestWendingOAuthSidebarApps(t *testing.T) {
	env := setupWendingOAuthTest(t, false)

	// Default client is not shown in the sidebar.
	req := httptest.NewRequest(http.MethodGet, "/api/wending/oauth/sidebar-apps", nil)
	rec, body := env.do(req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, true, body["success"])
	require.Len(t, body["data"].([]any), 0)

	// show_in_sidebar without launch_url must be rejected by apply.
	err := applyWendingOAuthClientRequest(env.client, wendingOAuthClientRequest{
		Name: "Huabu", RedirectUris: []string{wendingTestRedirect}, Enabled: true, ShowInSidebar: true,
	})
	require.Error(t, err)

	env.client.ShowInSidebar = true
	env.client.LaunchUrl = "https://huabu.wending.ai/"
	env.client.SidebarOrder = 10
	require.NoError(t, env.client.Save())

	hidden := &model.WendingOAuthClient{Name: "hidden", ClientId: "wd_hidden0000000000000000",
		Enabled: true, ShowInSidebar: false, LaunchUrl: "https://example.com/"}
	require.NoError(t, hidden.SetRedirectURIs([]string{wendingTestRedirect}))
	require.NoError(t, hidden.Insert())

	disabled := &model.WendingOAuthClient{Name: "disabled", ClientId: "wd_disabled0000000000000",
		Enabled: false, ShowInSidebar: true, LaunchUrl: "https://example.com/x", SidebarOrder: 1}
	require.NoError(t, disabled.SetRedirectURIs([]string{wendingTestRedirect}))
	require.NoError(t, disabled.Insert())

	earlier := &model.WendingOAuthClient{Name: "earlier", ClientId: "wd_earlier00000000000000",
		Enabled: true, ShowInSidebar: true, LaunchUrl: "https://earlier.example/", SidebarOrder: 5}
	require.NoError(t, earlier.SetRedirectURIs([]string{wendingTestRedirect}))
	require.NoError(t, earlier.Insert())

	rec, body = env.do(httptest.NewRequest(http.MethodGet, "/api/wending/oauth/sidebar-apps", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	apps := body["data"].([]any)
	require.Len(t, apps, 2)
	require.Equal(t, "earlier", apps[0].(map[string]any)["name"])
	require.Equal(t, "https://earlier.example/", apps[0].(map[string]any)["launch_url"])
	require.Equal(t, "WendingAI 创作台", apps[1].(map[string]any)["name"])
	require.Equal(t, "https://huabu.wending.ai/", apps[1].(map[string]any)["launch_url"])
}
