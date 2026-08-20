package common

import (
	"testing"

	rootcommon "github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCostSavingVisibleUsageUsesOriginalPromptTokens(t *testing.T) {
	info := &RelayInfo{
		CostSaving: &CostSavingInfo{
			Enabled:              true,
			OriginalPromptTokens: 120,
		},
	}
	usage := &dto.Usage{
		PromptTokens:         42,
		CompletionTokens:     18,
		TotalTokens:          60,
		PromptCacheHitTokens: 8,
		InputTokens:          42,
		OutputTokens:         18,
		InputTokensDetails:   &dto.InputTokenDetails{CachedTokens: 8},
		PromptTokensDetails:  dto.InputTokenDetails{CachedTokens: 8},
	}

	visible := CostSavingVisibleUsage(info, usage)

	require.NotNil(t, visible)
	assert.Equal(t, 120, visible.PromptTokens)
	assert.Equal(t, 18, visible.CompletionTokens)
	assert.Equal(t, 138, visible.TotalTokens)
	assert.Equal(t, 120, visible.InputTokens)
	assert.Equal(t, 18, visible.OutputTokens)
	assert.Zero(t, visible.PromptCacheHitTokens)
	assert.Zero(t, visible.PromptTokensDetails)
	assert.Nil(t, visible.InputTokensDetails)
}

func TestHideCostSavingModelInBodyRewritesModelAndUsage(t *testing.T) {
	info := &RelayInfo{
		CostSaving: &CostSavingInfo{
			Enabled:              true,
			HideResponseModel:    true,
			OriginalModelName:    "gpt-5",
			OriginalPromptTokens: 100,
		},
	}
	body := []byte(`{"id":"chatcmpl_1","model":"gpt-5.4-mini","usage":{"prompt_tokens":40,"completion_tokens":20,"total_tokens":60,"prompt_tokens_details":{"cached_tokens":10}}}`)

	out := HideCostSavingModelInBody(info, body)

	var payload map[string]interface{}
	require.NoError(t, rootcommon.Unmarshal(out, &payload))
	assert.Equal(t, "gpt-5", payload["model"])

	usage, ok := payload["usage"].(map[string]interface{})
	require.True(t, ok)
	assert.EqualValues(t, 100, usage["prompt_tokens"])
	assert.EqualValues(t, 20, usage["completion_tokens"])
	assert.EqualValues(t, 120, usage["total_tokens"])
	assert.Equal(t, map[string]interface{}{}, usage["prompt_tokens_details"])
}

func TestHideCostSavingModelInStreamRewritesUsageWithoutModel(t *testing.T) {
	info := &RelayInfo{
		CostSaving: &CostSavingInfo{
			Enabled:              true,
			HideResponseModel:    true,
			OriginalModelName:    "gpt-5",
			OriginalPromptTokens: 90,
		},
	}
	data := `{"choices":[],"usage":{"prompt_tokens":30,"completion_tokens":15,"total_tokens":45}}`

	out := HideCostSavingModelInStream(info, data)

	var payload map[string]interface{}
	require.NoError(t, rootcommon.Unmarshal([]byte(out), &payload))
	usage, ok := payload["usage"].(map[string]interface{})
	require.True(t, ok)
	assert.EqualValues(t, 90, usage["prompt_tokens"])
	assert.EqualValues(t, 105, usage["total_tokens"])
}
