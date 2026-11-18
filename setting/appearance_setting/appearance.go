package appearance_setting

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/setting/config"
)

// 主题模式
const (
	ThemeModeLight  = "light"
	ThemeModeDark   = "dark"
	ThemeModeSystem = "system"
)

// MaxThemePresetLength 主题预设名最大长度。预设样式由前端 CSS 定义，
// 后端只负责下发管理员选择的名称，因此不做白名单校验。
const MaxThemePresetLength = 64

type AppearanceSetting struct {
	// ThemeMode 全站主题模式：light / dark / system
	ThemeMode string `json:"theme_mode"`
	// ThemePreset 前端颜色预设名，例如 anthropic
	ThemePreset string `json:"theme_preset"`
	// AllowUserTheme 是否允许用户自行切换主题；false 时为全站强制主题
	AllowUserTheme bool `json:"allow_user_theme"`
}

var appearanceSetting = AppearanceSetting{
	ThemeMode:      ThemeModeSystem,
	ThemePreset:    "default",
	AllowUserTheme: true,
}

func init() {
	config.GlobalConfig.Register("appearance_setting", &appearanceSetting)
}

func GetAppearanceSetting() *AppearanceSetting {
	return &appearanceSetting
}

// GetThemeMode 返回归一化后的主题模式；脏数据回退到 system。
func GetThemeMode() string {
	if IsValidThemeMode(appearanceSetting.ThemeMode) {
		return appearanceSetting.ThemeMode
	}
	return ThemeModeSystem
}

func IsValidThemeMode(mode string) bool {
	switch mode {
	case ThemeModeLight, ThemeModeDark, ThemeModeSystem:
		return true
	default:
		return false
	}
}

// ValidateThemePreset 校验前端预设名。允许为空（前端回退默认预设）。
func ValidateThemePreset(preset string) error {
	if len(preset) > MaxThemePresetLength {
		return errors.New("theme preset is too long")
	}
	if strings.TrimSpace(preset) != preset {
		return errors.New("theme preset must not contain surrounding whitespace")
	}
	return nil
}
