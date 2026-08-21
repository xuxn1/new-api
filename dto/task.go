package dto

import (
	"encoding/json"
)

type TaskError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	Data       any    `json:"data"`
	StatusCode int    `json:"-"`
	LocalError bool   `json:"-"`
	Error      error  `json:"-"`
}

type TaskData interface {
	SunoDataResponse | []SunoDataResponse | string | any
}

const TaskSuccessCode = "success"

type TaskResponse[T TaskData] struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

func (t *TaskResponse[T]) IsSuccess() bool {
	return t.Code == TaskSuccessCode
}

type TaskDto struct {
	ID                int64                  `json:"id"`
	CreatedAt         int64                  `json:"created_at"`
	UpdatedAt         int64                  `json:"updated_at"`
	TaskID            string                 `json:"task_id"`
	Platform          string                 `json:"platform"`
	UserId            int                    `json:"user_id"`
	Group             string                 `json:"group"`
	ChannelId         int                    `json:"channel_id"`
	Quota             int                    `json:"quota"`
	Action            string                 `json:"action"`
	Status            string                 `json:"status"`
	FailReason        string                 `json:"fail_reason"`
	ResultURL         string                 `json:"result_url,omitempty"` // 任务结果 URL（视频地址等）
	SubmitTime        int64                  `json:"submit_time"`
	StartTime         int64                  `json:"start_time"`
	FinishTime        int64                  `json:"finish_time"`
	Progress          string                 `json:"progress"`
	Properties        any                    `json:"properties"`
	Username          string                 `json:"username,omitempty"`
	CostSavingContext *TaskCostSavingContext `json:"cost_saving_context,omitempty"`
	Data              json.RawMessage        `json:"data"`
}

type TaskCostSavingContext struct {
	Enabled                   bool   `json:"enabled,omitempty"`
	RuleName                  string `json:"rule_name,omitempty"`
	Strategy                  string `json:"strategy,omitempty"`
	OriginalModelName         string `json:"original_model_name,omitempty"`
	PlannerModelName          string `json:"planner_model_name,omitempty"`
	ExecutorModelName         string `json:"executor_model_name,omitempty"`
	PlannerUpstreamModelName  string `json:"planner_upstream_model_name,omitempty"`
	ExecutorUpstreamModelName string `json:"executor_upstream_model_name,omitempty"`
	ActualUpstreamModelName   string `json:"actual_upstream_model_name,omitempty"`
	PlannerChannelId          int    `json:"planner_channel_id,omitempty"`
	PlannerChannelName        string `json:"planner_channel_name,omitempty"`
	PlannerChannelType        int    `json:"planner_channel_type,omitempty"`
	ExecutorChannelId         int    `json:"executor_channel_id,omitempty"`
	ExecutorChannelName       string `json:"executor_channel_name,omitempty"`
	ExecutorChannelType       int    `json:"executor_channel_type,omitempty"`
	AnalysisInjected          bool   `json:"analysis_injected,omitempty"`
	CacheHit                  bool   `json:"cache_hit,omitempty"`
	CacheScope                string `json:"cache_scope,omitempty"`
	FallbackUsed              bool   `json:"fallback_used,omitempty"`
	FallbackReason            string `json:"fallback_reason,omitempty"`
	HideResponseModel         bool   `json:"hide_response_model,omitempty"`
	OriginalPromptTokens      int    `json:"original_prompt_tokens,omitempty"`
	PlannerPromptTokens       int    `json:"planner_prompt_tokens,omitempty"`
	PlannerCompletionTokens   int    `json:"planner_completion_tokens,omitempty"`
	ExecutorPromptTokens      int    `json:"executor_prompt_tokens,omitempty"`
	ExecutorCompletionTokens  int    `json:"executor_completion_tokens,omitempty"`
	RawPromptTokens           int    `json:"raw_prompt_tokens,omitempty"`
	RawCompletionTokens       int    `json:"raw_completion_tokens,omitempty"`
	RawTotalTokens            int    `json:"raw_total_tokens,omitempty"`
	PlannerEstimatedQuota     int    `json:"planner_estimated_quota,omitempty"`
	ExecutorEstimatedQuota    int    `json:"executor_estimated_quota,omitempty"`
	ActualEstimatedQuota      int    `json:"actual_estimated_quota,omitempty"`
	OriginalBilledQuota       int    `json:"original_billed_quota,omitempty"`
	SavingQuota               int    `json:"saving_quota,omitempty"`
}

type FetchReq struct {
	IDs []string `json:"ids"`
}
