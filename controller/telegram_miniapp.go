package controller

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/oauth"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Telegram Mini App login (POST /api/oauth/telegram/miniapp/login).
//
// Unlike the Telegram OIDC login, which requires a ClientID/ClientSecret and
// refuses users whose Telegram account is not already bound, a Mini App
// opened from Telegram carries a signed initData payload verified with the
// bot token (common.TelegramBotToken, already loaded from the DB options
// table). A verified payload may auto-create the user, the same way the
// shop (dujiao-wending) creates a Telegram user on first Mini App login.
// The OIDC flow is untouched.

type telegramMiniAppLoginRequest struct {
	InitData    string `json:"init_data"`
	InitDataAlt string `json:"initData"`
}

// TelegramMiniAppLogin verifies a Telegram WebApp initData payload and
// issues the normal login session for the Telegram user, creating the user
// on first login.
func TelegramMiniAppLogin(c *gin.Context) {
	botToken := strings.TrimSpace(common.TelegramBotToken)
	if botToken == "" {
		writeSecurityOperationError(c, oauth.ErrTelegramMiniAppNotConfigured)
		return
	}
	var request telegramMiniAppLoginRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		common.ApiErrorT(c, "Invalid parameters")
		return
	}
	initData := strings.TrimSpace(request.InitData)
	if initData == "" {
		initData = strings.TrimSpace(request.InitDataAlt)
	}
	if initData == "" {
		common.ApiErrorT(c, "Invalid parameters")
		return
	}
	auth, err := oauth.VerifyTelegramMiniAppInitData(initData, botToken, time.Now())
	if err != nil {
		writeSecurityOperationError(c, err)
		return
	}
	// The initData hash is single-use: claim it before any state change so
	// a replayed payload is rejected even if the login itself later fails.
	// Telegram issues a fresh initData (new auth_date/hash) every time the
	// Mini App is reopened, so a legitimate retry is not blocked.
	if err := model.ClaimExternalAuthAssertion(
		model.AuthFlowPurposeTelegramAssertion,
		auth.Hash,
		auth.AuthDate.Add(oauth.TelegramMiniAppAuthValidity),
	); err != nil {
		if errors.Is(err, model.ErrAuthFlowConsumed) {
			writeSecurityOperationError(c, oauth.ErrTelegramMiniAppFailed)
			return
		}
		writeSecurityOperationError(c, err)
		return
	}
	user, err := findOrCreateTelegramMiniAppUser(auth)
	if err != nil {
		switch err.(type) {
		case *OAuthRegistrationDisabledError:
			common.ApiErrorT(c, "New user registration has been disabled by administrator")
		default:
			writeSecurityOperationError(c, err)
		}
		return
	}
	if user.Status != common.UserStatusEnabled {
		common.ApiErrorT(c, "User has been banned")
		return
	}
	setupLogin(user, nil, c)
}

// findOrCreateTelegramMiniAppUser returns the user bound to the verified
// Telegram id, creating it when registration is enabled. The creation path
// mirrors findOrCreateOAuthUser for built-in providers: same username
// fallback, same empty-password/empty-email Telegram profile, same
// transactional insert plus provider-id update and post-creation finalize.
func findOrCreateTelegramMiniAppUser(auth *oauth.TelegramMiniAppAuth) (*model.User, error) {
	telegramID := strconv.FormatInt(auth.User.ID, 10)
	user, err := model.GetUserByTelegramID(telegramID)
	if err == nil {
		return user, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if !common.RegisterEnabled {
		return nil, &OAuthRegistrationDisabledError{}
	}
	user = &model.User{}
	user.Username = "telegram_" + strconv.Itoa(model.GetMaxUserId()+1)
	if auth.User.Username != "" && len(auth.User.Username) <= model.UserNameMaxLength {
		if exists, err := model.CheckUserExistOrDeleted(auth.User.Username, ""); err == nil && !exists {
			user.Username = auth.User.Username
		}
	}
	displayName := strings.TrimSpace(strings.Join([]string{auth.User.FirstName, auth.User.LastName}, " "))
	if displayName == "" {
		displayName = "Telegram User"
	}
	user.DisplayName = truncateTelegramMiniAppDisplayName(displayName, model.UserNameMaxLength)
	user.Role = common.RoleCommonUser
	user.Status = common.UserStatusEnabled
	if err := model.DB.Transaction(func(tx *gorm.DB) error {
		if err := user.InsertWithTx(tx, 0); err != nil {
			return err
		}
		user.TelegramId = telegramID
		return tx.Model(user).Updates(map[string]any{
			"github_id":   user.GitHubId,
			"discord_id":  user.DiscordId,
			"oidc_id":     user.OidcId,
			"linux_do_id": user.LinuxDOId,
			"wechat_id":   user.WeChatId,
			"telegram_id": user.TelegramId,
		}).Error
	}); err != nil {
		return nil, err
	}
	user.FinalizeOAuthUserCreation(0)
	return user, nil
}

// truncateTelegramMiniAppDisplayName shortens value to at most max bytes
// without splitting a UTF-8 rune, matching the byte-length validation used
// for user names.
func truncateTelegramMiniAppDisplayName(value string, max int) string {
	if len(value) <= max {
		return value
	}
	runes := []rune(value)
	for len(runes) > 0 && len(string(runes)) > max {
		runes = runes[:len(runes)-1]
	}
	return string(runes)
}
