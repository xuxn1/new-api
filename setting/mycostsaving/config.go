package mycostsaving

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

const (
	ConfigName = "my_cost_saving"
)

type Rule struct {
	Enabled       bool     `json:"enabled"`
	Name          string   `json:"name"`
	Groups        []string `json:"groups"`
	Models        []string `json:"models"`
	Strategy      string   `json:"strategy,omitempty"`
	AnalysisModel string   `json:"analysis_model,omitempty"`
	// PlannerModel is kept for backwards compatibility with the original rule schema.
	PlannerModel string `json:"planner_model,omitempty"`
	// ExecutorModel and ComplexModel are legacy final-model overrides. New rules
	// always execute on the model requested by the customer.
	ExecutorModel    string `json:"executor_model,omitempty"`
	ComplexModel     string `json:"complex_model,omitempty"`
	MaxLowCostTokens int    `json:"max_low_cost_tokens,omitempty"`
	CacheEnabled     *bool  `json:"cache_enabled,omitempty"`
	CacheTTLSeconds  int    `json:"cache_ttl_seconds,omitempty"`
	CacheScope       string `json:"cache_scope,omitempty"`
}

type Settings struct {
	Enabled                 bool   `json:"enabled"`
	RulesJSON               string `json:"rules_json"`
	InjectAnalysisToRequest bool   `json:"inject_analysis_to_request"`
	FallbackToOriginal      bool   `json:"fallback_to_original"`
	DisableForStream        bool   `json:"disable_for_stream"`
	HideResponseModel       bool   `json:"hide_response_model"`
	MaxPlannerTokens        int    `json:"max_planner_tokens"`
	PlannerPrompt           string `json:"planner_prompt"`
	ExactCacheEnabled       bool   `json:"exact_cache_enabled"`
	ExactCacheTTLSeconds    int    `json:"exact_cache_ttl_seconds"`
	MaxLowCostPromptTokens  int    `json:"max_low_cost_prompt_tokens"`
}

type Match struct {
	Rule             Rule
	Strategy         string
	AnalysisModel    string
	PlannerModel     string
	ExecutorModel    string
	ComplexModel     string
	LegacyExecution  bool
	MaxLowCostTokens int
	CacheEnabled     bool
	CacheTTLSeconds  int
	CacheScope       string
}

var defaultSettings = Settings{
	Enabled:                 false,
	RulesJSON:               "[]",
	InjectAnalysisToRequest: true,
	FallbackToOriginal:      true,
	DisableForStream:        true,
	HideResponseModel:       true,
	MaxPlannerTokens:        512,
	PlannerPrompt:           defaultPlannerPrompt,
	ExactCacheEnabled:       true,
	ExactCacheTTLSeconds:    600,
	MaxLowCostPromptTokens:  2000,
}

const defaultPlannerPrompt = "Analyze the user's request, identify the concrete work to do, and return a concise execution plan. Do not answer the user directly."
const maxPlannerTokensLimit = math.MaxInt32 / 2
const maxCacheTTLSecondsLimit = 86400 * 30
const maxLowCostPromptTokensLimit = math.MaxInt32 / 2

func init() {
	config.GlobalConfig.Register(ConfigName, &defaultSettings)
}

func GetSettings() Settings {
	return defaultSettings
}

func ValidateRulesJSON(value string) error {
	_, err := parseRules(value)
	return err
}

func ValidateMaxPlannerTokens(value string) error {
	return validateBoundedInt(value, "max_planner_tokens", 0, maxPlannerTokensLimit)
}

func ValidateExactCacheTTLSeconds(value string) error {
	return validateBoundedInt(value, "exact_cache_ttl_seconds", 0, maxCacheTTLSecondsLimit)
}

func ValidateMaxLowCostPromptTokens(value string) error {
	return validateBoundedInt(value, "max_low_cost_prompt_tokens", 0, maxLowCostPromptTokensLimit)
}

func validateBoundedInt(value string, name string, minValue int, maxValue int) error {
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return err
	}
	if n < minValue || n > maxValue {
		return fmt.Errorf("%s must be between %d and %d", name, minValue, maxValue)
	}
	return nil
}

func MatchRule(group string, modelName string, isStream bool) (Match, bool) {
	settings := GetSettings()
	if !settings.Enabled {
		return Match{}, false
	}
	if settings.DisableForStream && isStream {
		return Match{}, false
	}

	rules, err := parseRules(settings.RulesJSON)
	if err != nil {
		common.SysError("invalid my_cost_saving rules_json: " + err.Error())
		return Match{}, false
	}

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		strategy := normalizeStrategy(rule.Strategy)
		analysisModel := strings.TrimSpace(rule.AnalysisModel)
		if analysisModel == "" {
			analysisModel = strings.TrimSpace(rule.PlannerModel)
		}
		legacyExecutorModel := strings.TrimSpace(rule.ExecutorModel)
		legacyExecution := legacyExecutorModel != ""
		if legacyExecution && analysisModel == "" {
			if strategy == "planner" {
				analysisModel = modelName
			} else {
				analysisModel = legacyExecutorModel
			}
		}
		if !matchesAny(group, rule.Groups) {
			continue
		}
		if !matchesModel(modelName, rule.Models) {
			continue
		}
		cacheEnabled := settings.ExactCacheEnabled
		if rule.CacheEnabled != nil {
			cacheEnabled = *rule.CacheEnabled
		}
		cacheTTLSeconds := settings.ExactCacheTTLSeconds
		if rule.CacheTTLSeconds > 0 {
			cacheTTLSeconds = rule.CacheTTLSeconds
		}
		maxLowCostTokens := settings.MaxLowCostPromptTokens
		if rule.MaxLowCostTokens > 0 {
			maxLowCostTokens = rule.MaxLowCostTokens
		}
		executorModel := modelName
		if legacyExecution {
			executorModel = legacyExecutorModel
		}
		return Match{
			Rule:             rule,
			Strategy:         strategy,
			AnalysisModel:    analysisModel,
			PlannerModel:     analysisModel,
			ExecutorModel:    executorModel,
			ComplexModel:     strings.TrimSpace(rule.ComplexModel),
			LegacyExecution:  legacyExecution,
			MaxLowCostTokens: maxLowCostTokens,
			CacheEnabled:     cacheEnabled,
			CacheTTLSeconds:  cacheTTLSeconds,
			CacheScope:       normalizeCacheScope(rule.CacheScope),
		}, true
	}

	return Match{}, false
}

func normalizeStrategy(strategy string) string {
	switch strings.ToLower(strings.TrimSpace(strategy)) {
	case "planner":
		return "planner"
	case "auto":
		return "auto"
	default:
		return "direct"
	}
}

func normalizeCacheScope(scope string) string {
	switch strings.ToLower(strings.TrimSpace(scope)) {
	case "user":
		return "user"
	default:
		return "group"
	}
}

func parseRules(value string) ([]Rule, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	var rules []Rule
	if err := common.Unmarshal([]byte(value), &rules); err != nil {
		return nil, err
	}
	return rules, nil
}

func matchesAny(value string, candidates []string) bool {
	if len(candidates) == 0 {
		return true
	}
	value = strings.TrimSpace(value)
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" || candidate == "*" {
			return true
		}
		if candidate == value {
			return true
		}
	}
	return false
}

func matchesModel(modelName string, candidates []string) bool {
	if len(candidates) == 0 {
		return true
	}
	formattedModel := ratio_setting.FormatMatchingModelName(modelName)
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" || candidate == "*" {
			return true
		}
		if strings.HasSuffix(candidate, "*") {
			prefix := strings.TrimSuffix(candidate, "*")
			if strings.HasPrefix(modelName, prefix) || strings.HasPrefix(formattedModel, prefix) {
				return true
			}
			continue
		}
		if candidate == modelName || ratio_setting.FormatMatchingModelName(candidate) == formattedModel {
			return true
		}
	}
	return false
}
