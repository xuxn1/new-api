package common

import (
	rootcommon "github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
)

func CostSavingBillingUsage(info *RelayInfo, usage *dto.Usage) *dto.Usage {
	if usage == nil {
		return nil
	}
	adjusted := *usage
	if info == nil || info.CostSaving == nil || !info.CostSaving.Enabled {
		return &adjusted
	}
	adjusted.PromptTokens = info.CostSaving.OriginalPromptTokens
	adjusted.TotalTokens = adjusted.PromptTokens + adjusted.CompletionTokens
	adjusted.BillingUsage = nil
	return &adjusted
}

func CostSavingVisibleUsage(info *RelayInfo, usage *dto.Usage) *dto.Usage {
	if usage == nil {
		return nil
	}
	adjusted := *CostSavingBillingUsage(info, usage)
	if info == nil || info.CostSaving == nil || !info.CostSaving.Enabled {
		return &adjusted
	}
	if adjusted.InputTokens > 0 {
		adjusted.InputTokens = adjusted.PromptTokens
	}
	if adjusted.OutputTokens > 0 {
		adjusted.OutputTokens = adjusted.CompletionTokens
	}
	adjusted.PromptTokensDetails = dto.InputTokenDetails{}
	adjusted.InputTokensDetails = nil
	adjusted.PromptCacheHitTokens = 0
	return &adjusted
}

func HideCostSavingModelInStream(info *RelayInfo, data string) string {
	if info == nil || info.CostSaving == nil || !info.CostSaving.HideResponseModel || info.CostSaving.OriginalModelName == "" || data == "" {
		return data
	}
	var payload map[string]interface{}
	if err := rootcommon.Unmarshal([]byte(data), &payload); err != nil {
		return data
	}
	if _, ok := payload["model"]; !ok {
		if !rewriteCostSavingUsagePayload(info, payload) {
			return data
		}
		out, err := rootcommon.Marshal(payload)
		if err != nil {
			return data
		}
		return string(out)
	}
	payload["model"] = info.CostSaving.OriginalModelName
	rewriteCostSavingUsagePayload(info, payload)
	out, err := rootcommon.Marshal(payload)
	if err != nil {
		return data
	}
	return string(out)
}

func HideCostSavingModelInBody(info *RelayInfo, data []byte) []byte {
	if info == nil || info.CostSaving == nil || !info.CostSaving.HideResponseModel || info.CostSaving.OriginalModelName == "" || len(data) == 0 {
		return data
	}
	var payload map[string]interface{}
	if err := rootcommon.Unmarshal(data, &payload); err != nil {
		return data
	}
	if _, ok := payload["model"]; !ok {
		if !rewriteCostSavingUsagePayload(info, payload) {
			return data
		}
		out, err := rootcommon.Marshal(payload)
		if err != nil {
			return data
		}
		return out
	}
	payload["model"] = info.CostSaving.OriginalModelName
	rewriteCostSavingUsagePayload(info, payload)
	out, err := rootcommon.Marshal(payload)
	if err != nil {
		return data
	}
	return out
}

func rewriteCostSavingUsagePayload(info *RelayInfo, payload map[string]interface{}) bool {
	if info == nil || info.CostSaving == nil || !info.CostSaving.Enabled || payload == nil {
		return false
	}
	usage, ok := payload["usage"].(map[string]interface{})
	if !ok || usage == nil {
		return false
	}
	completionTokens := usageNumberToInt(usage["completion_tokens"])
	if completionTokens == 0 {
		completionTokens = usageNumberToInt(usage["output_tokens"])
	}
	promptTokens := info.CostSaving.OriginalPromptTokens
	usage["prompt_tokens"] = promptTokens
	usage["total_tokens"] = promptTokens + completionTokens
	if _, ok := usage["input_tokens"]; ok {
		usage["input_tokens"] = promptTokens
	}
	if _, ok := usage["output_tokens"]; ok {
		usage["output_tokens"] = completionTokens
	}
	delete(usage, "billing_usage")
	if _, ok := usage["prompt_tokens_details"]; ok {
		usage["prompt_tokens_details"] = map[string]interface{}{}
	}
	if _, ok := usage["input_tokens_details"]; ok {
		usage["input_tokens_details"] = map[string]interface{}{}
	}
	return true
}

func usageNumberToInt(value interface{}) int {
	switch v := value.(type) {
	case int:
		if v > 0 {
			return v
		}
	case int64:
		if v > 0 && v <= int64(maxIntValue()) {
			return int(v)
		}
	case float64:
		if v > 0 && v <= float64(maxIntValue()) {
			return int(v)
		}
	case uint64:
		if v > 0 && v <= uint64(maxIntValue()) {
			return int(v)
		}
	}
	return 0
}

func maxIntValue() int {
	return int(^uint(0) >> 1)
}
