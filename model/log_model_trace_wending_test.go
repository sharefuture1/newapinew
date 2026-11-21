package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 稳定AI: model-routing diagnostics (mapped upstream model and the model
// name the upstream returned) are visible to root only.
func TestModelTraceLogOtherIsRootOnly(t *testing.T) {
	other := common.MapToJsonStr(map[string]any{
		"model_price":         0.5,
		"is_model_mapped":     true,
		"upstream_model_name": "upstream-x",
		"response_model": map[string]any{
			"requested_model": "kimi-k3",
			"upstream_model":  "kimi-k3",
			"returned_model":  "other-model",
		},
	})
	traceKeys := []string{"is_model_mapped", "upstream_model_name", "response_model"}

	for name, format := range map[string]func([]*Log){
		"user":  func(logs []*Log) { formatUserLogs(logs, 0) },
		"admin": FormatAdminLogs,
	} {
		t.Run(name, func(t *testing.T) {
			logs := []*Log{{Other: other}}
			format(logs)
			parsed, err := common.StrToMap(logs[0].Other)
			require.NoError(t, err)
			for _, key := range traceKeys {
				assert.NotContains(t, parsed, key)
			}
			assert.Equal(t, 0.5, parsed["model_price"])
		})
	}

	t.Run("root", func(t *testing.T) {
		logs := []*Log{{Other: other}}
		FormatRootLogs(logs)
		parsed, err := common.StrToMap(logs[0].Other)
		require.NoError(t, err)
		for _, key := range traceKeys {
			assert.Contains(t, parsed, key)
		}
	})

	t.Run("user export stream shape", func(t *testing.T) {
		assert.NotContains(t, formatLogOtherJSON(other, logOtherVisibilityUser), "upstream-x")
		assert.NotContains(t, formatLogOtherJSON(other, logOtherVisibilityAdmin), "other-model")
		assert.Contains(t, formatLogOtherJSON(other, logOtherVisibilityRoot), "other-model")
	})
}
