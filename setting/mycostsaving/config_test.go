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
			"planner_model": "gpt-5.4-mini",
			"executor_model": "gpt-5.4-mini"
		}
	]`

	match, ok := MatchRule("vip", "gpt-5", false)
	require.True(t, ok)
	assert.Equal(t, "vip gpt-5", match.Rule.Name)
	assert.Equal(t, "gpt-5.4-mini", match.PlannerModel)
	assert.Equal(t, "gpt-5.4-mini", match.ExecutorModel)

	_, ok = MatchRule("default", "gpt-5", false)
	assert.False(t, ok)

	_, ok = MatchRule("vip", "gpt-5", true)
	assert.False(t, ok)
}
