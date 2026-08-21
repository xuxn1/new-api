package mycostsaving

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateMaxPlannerTokens(t *testing.T) {
	require.NoError(t, ValidateMaxPlannerTokens("0"))
	require.NoError(t, ValidateMaxPlannerTokens("512"))

	assert.Error(t, ValidateMaxPlannerTokens("-1"))
	assert.Error(t, ValidateMaxPlannerTokens(strconv.Itoa(maxPlannerTokensLimit+1)))
	assert.Error(t, ValidateMaxPlannerTokens("not-a-number"))
}

func TestValidateCostSavingIntegerSettings(t *testing.T) {
	require.NoError(t, ValidateExactCacheTTLSeconds("600"))
	require.NoError(t, ValidateMaxLowCostPromptTokens("2000"))

	assert.Error(t, ValidateExactCacheTTLSeconds("-1"))
	assert.Error(t, ValidateExactCacheTTLSeconds(strconv.Itoa(maxCacheTTLSecondsLimit+1)))
	assert.Error(t, ValidateMaxLowCostPromptTokens("-1"))
	assert.Error(t, ValidateMaxLowCostPromptTokens(strconv.Itoa(maxLowCostPromptTokensLimit+1)))
}

func TestMatchRuleMatchesGroupAndModel(t *testing.T) {
	oldSettings := defaultSettings
	defer func() {
		defaultSettings = oldSettings
	}()

	defaultSettings.Enabled = true
	defaultSettings.DisableForStream = true
	defaultSettings.RulesJSON = `[
		{
			"enabled": true,
			"name": "vip gpt-5",
			"groups": ["vip"],
			"models": ["gpt-5*"],
			"strategy": "auto",
			"executor_model": "gpt-5.4-mini"
		}
	]`

	match, ok := MatchRule("vip", "gpt-5", false)
	require.True(t, ok)
	assert.Equal(t, "vip gpt-5", match.Rule.Name)
	assert.Equal(t, "auto", match.Strategy)
	assert.Equal(t, "gpt-5.4-mini", match.PlannerModel)
	assert.Equal(t, "gpt-5.4-mini", match.ExecutorModel)
	assert.True(t, match.LegacyExecution)
	assert.True(t, match.CacheEnabled)
	assert.Equal(t, 600, match.CacheTTLSeconds)

	_, ok = MatchRule("default", "gpt-5", false)
	assert.False(t, ok)

	_, ok = MatchRule("vip", "gpt-5", true)
	assert.False(t, ok)
}

func TestMatchRuleDefaultsToDirectStrategy(t *testing.T) {
	oldSettings := defaultSettings
	defer func() {
		defaultSettings = oldSettings
	}()

	defaultSettings.Enabled = true
	defaultSettings.DisableForStream = false
	defaultSettings.RulesJSON = `[{"enabled":true,"models":["gpt-5"],"executor_model":"gpt-5.4-mini"}]`

	match, ok := MatchRule("default", "gpt-5", false)
	require.True(t, ok)
	assert.Equal(t, "direct", match.Strategy)
	assert.Equal(t, "gpt-5.4-mini", match.PlannerModel)
	assert.Equal(t, "gpt-5.4-mini", match.ExecutorModel)
	assert.True(t, match.LegacyExecution)
}

func TestMatchRulePlannerFallsBackToOriginalModel(t *testing.T) {
	oldSettings := defaultSettings
	defer func() {
		defaultSettings = oldSettings
	}()

	defaultSettings.Enabled = true
	defaultSettings.DisableForStream = false
	defaultSettings.RulesJSON = `[{"enabled":true,"strategy":"planner","models":["gpt-5"],"executor_model":"gpt-5.4-mini"}]`

	match, ok := MatchRule("default", "gpt-5", false)
	require.True(t, ok)
	assert.Equal(t, "planner", match.Strategy)
	assert.Equal(t, "gpt-5", match.PlannerModel)
	assert.Equal(t, "gpt-5.4-mini", match.ExecutorModel)
	assert.True(t, match.LegacyExecution)
}

func TestMatchRuleUsesOriginalModelForNewRules(t *testing.T) {
	oldSettings := defaultSettings
	defer func() {
		defaultSettings = oldSettings
	}()

	defaultSettings.Enabled = true
	defaultSettings.DisableForStream = false
	defaultSettings.RulesJSON = `[{"enabled":true,"strategy":"planner","models":["gpt-5"],"analysis_model":"gpt-5.4-mini"}]`

	match, ok := MatchRule("default", "gpt-5", false)
	require.True(t, ok)
	assert.Equal(t, "planner", match.Strategy)
	assert.Equal(t, "gpt-5.4-mini", match.AnalysisModel)
	assert.Equal(t, "gpt-5.4-mini", match.PlannerModel)
	assert.Equal(t, "gpt-5", match.ExecutorModel)
	assert.False(t, match.LegacyExecution)
}
