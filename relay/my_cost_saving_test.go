package relay

import (
	"testing"

	"github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildTaskCostSavingContext(t *testing.T) {
	costInfo := &common.CostSavingInfo{
		Enabled:                  true,
		RuleName:                 "vip gpt-5",
		OriginalModelName:        "gpt-5",
		PlannerModelName:         "gpt-5.4-mini",
		ExecutorModelName:        "gpt-5.4-mini",
		PlannerChannelName:       "planner-a",
		ExecutorChannelName:      "executor-b",
		OriginalPromptTokens:     120,
		PlannerPromptTokens:      16,
		PlannerCompletionTokens:  8,
		ExecutorPromptTokens:     42,
		ExecutorCompletionTokens: 64,
		RawPromptTokens:          58,
		RawCompletionTokens:      72,
		RawTotalTokens:           130,
		PlannerEstimatedQuota:    30,
		ExecutorEstimatedQuota:   40,
		ActualEstimatedQuota:     70,
		OriginalBilledQuota:      96,
	}

	ctx := BuildTaskCostSavingContext(costInfo)
	require.NotNil(t, ctx)
	assert.Equal(t, "vip gpt-5", ctx.RuleName)
	assert.Equal(t, "gpt-5", ctx.OriginalModelName)
	assert.Equal(t, "gpt-5.4-mini", ctx.PlannerModelName)
	assert.Equal(t, "gpt-5.4-mini", ctx.ExecutorModelName)
	assert.Equal(t, 96, ctx.OriginalBilledQuota)
	assert.Equal(t, 26, ctx.SavingQuota)
}

func TestBuildTaskCostSavingContextClampsSavingQuota(t *testing.T) {
	costInfo := &common.CostSavingInfo{
		OriginalBilledQuota:  20,
		ActualEstimatedQuota: 35,
	}

	ctx := BuildTaskCostSavingContext(costInfo)
	require.NotNil(t, ctx)
	assert.Equal(t, 0, ctx.SavingQuota)
}

func TestBuildTaskCostSavingContextFallbackHasNoSaving(t *testing.T) {
	costInfo := &common.CostSavingInfo{
		FallbackUsed:         true,
		OriginalBilledQuota:  96,
		ActualEstimatedQuota: 0,
	}

	ctx := BuildTaskCostSavingContext(costInfo)
	require.NotNil(t, ctx)
	assert.Equal(t, 96, ctx.ActualEstimatedQuota)
	assert.Equal(t, 0, ctx.SavingQuota)
}
