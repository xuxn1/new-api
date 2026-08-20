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
	PlannerModel  string   `json:"planner_model,omitempty"`
	ExecutorModel string   `json:"executor_model"`
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
}

type Match struct {
	Rule          Rule
	PlannerModel  string
	ExecutorModel string
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
}

const defaultPlannerPrompt = "Analyze the user's request, identify the concrete work to do, and return a concise execution plan. Do not answer the user directly."
const maxPlannerTokensLimit = math.MaxInt32 / 2

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
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return err
	}
	if n < 0 || n > maxPlannerTokensLimit {
		return fmt.Errorf("max_planner_tokens must be between 0 and %d", maxPlannerTokensLimit)
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
		plannerModel := strings.TrimSpace(rule.PlannerModel)
		executorModel := strings.TrimSpace(rule.ExecutorModel)
		if plannerModel == "" {
			plannerModel = modelName
		}
		if executorModel == "" {
			continue
		}
		if !matchesAny(group, rule.Groups) {
			continue
		}
		if !matchesModel(modelName, rule.Models) {
			continue
		}
		return Match{
			Rule:          rule,
			PlannerModel:  plannerModel,
			ExecutorModel: executorModel,
		}, true
	}

	return Match{}, false
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
