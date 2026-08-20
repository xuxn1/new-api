package relay

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/QuantumNous/new-api/setting/mycostsaving"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/gin-gonic/gin"
)

type myCostSavingContextSnapshot struct {
	channelId             int
	channelName           string
	channelType           int
	channelCreateTime     int64
	channelSetting        dto.ChannelSettings
	channelOtherSetting   dto.ChannelOtherSettings
	paramOverride         map[string]interface{}
	headerOverride        map[string]interface{}
	organization          string
	autoBan               bool
	modelMapping          string
	statusCodeMapping     string
	channelIsMultiKey     bool
	channelMultiKeyIndex  int
	channelKey            string
	channelBaseUrl        string
	systemPromptOverwrite bool
	apiVersion            string
	region                string
	plugin                string
	botId                 string
	useChannel            []string
}

func maybeApplyMyCostSaving(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest, originalPromptTokens int) (bool, *types.NewAPIError) {
	match, ok := mycostsaving.MatchRule(info.UsingGroup, info.OriginModelName, info.IsStream)
	if !ok || request == nil {
		return false, nil
	}
	if !isMyCostSavingTextRequestSupported(request) {
		return false, nil
	}
	if model_setting.GetGlobalSettings().PassThroughRequestEnabled || info.ChannelSetting.PassThroughBodyEnabled {
		return false, nil
	}

	settings := mycostsaving.GetSettings()
	costInfo := &relaycommon.CostSavingInfo{
		Enabled:              true,
		RuleName:             match.Rule.Name,
		OriginalModelName:    info.OriginModelName,
		PlannerModelName:     match.PlannerModel,
		ExecutorModelName:    match.ExecutorModel,
		HideResponseModel:    true,
		OriginalPromptTokens: originalPromptTokens,
		OriginalBilledQuota:  info.FinalPreConsumedQuota,
	}
	info.CostSaving = costInfo

	analysis := ""
	if settings.InjectAnalysisToRequest {
		var plannerUsage *dto.Usage
		var plannerRunInfo *relaycommon.RelayInfo
		var err error
		analysis, plannerUsage, plannerRunInfo, err = runMyCostSavingPlanner(c, info, request, match.PlannerModel, settings.MaxPlannerTokens, settings.PlannerPrompt)
		if err != nil {
			return handleMyCostSavingFallback(c, info, fmt.Sprintf("planner failed: %s", err.Error()), nil)
		}
		applyMyCostSavingAttemptInfo(costInfo, plannerRunInfo, true)
		if plannerUsage != nil {
			costInfo.PlannerPromptTokens = plannerUsage.PromptTokens
			costInfo.PlannerCompletionTokens = plannerUsage.CompletionTokens
			costInfo.PlannerEstimatedQuota = estimateMyCostSavingQuota(match.PlannerModel, plannerUsage.PromptTokens, plannerUsage.CompletionTokens, info.PriceData.GroupRatioInfo.GroupRatio)
		}
	}

	executionRequest, err := common.DeepCopy(request)
	if err != nil {
		return handleMyCostSavingFallback(c, info, fmt.Sprintf("copy executor request failed: %s", err.Error()), nil)
	}
	if analysis != "" {
		executionRequest.Messages = appendMyCostSavingAnalysis(executionRequest.Messages, analysis)
		costInfo.AnalysisInjected = true
	}

	originalEstimateTokens := info.GetEstimatePromptTokens()
	executionRequest.SetModelName(match.ExecutorModel)
	info.SetEstimatePromptTokens(originalEstimateTokens + costInfo.PlannerCompletionTokens)
	usage, executionRunInfo, newAPIError := runMyCostSavingExecution(c, info, executionRequest, match.ExecutorModel)
	info.SetEstimatePromptTokens(originalEstimateTokens)
	if newAPIError != nil {
		return handleMyCostSavingFallback(c, info, newAPIError.Error(), newAPIError)
	}
	applyMyCostSavingAttemptInfo(costInfo, executionRunInfo, false)
	applyMyCostSavingExecutionInfo(info, executionRunInfo)

	if usage == nil {
		usage = &dto.Usage{}
	}
	costInfo.ExecutorPromptTokens = usage.PromptTokens
	costInfo.ExecutorCompletionTokens = usage.CompletionTokens
	costInfo.RawPromptTokens = costInfo.PlannerPromptTokens + costInfo.ExecutorPromptTokens
	costInfo.RawCompletionTokens = costInfo.PlannerCompletionTokens + costInfo.ExecutorCompletionTokens
	costInfo.RawTotalTokens = costInfo.RawPromptTokens + costInfo.RawCompletionTokens
	costInfo.ExecutorEstimatedQuota = estimateMyCostSavingQuota(match.ExecutorModel, usage.PromptTokens, usage.CompletionTokens, info.PriceData.GroupRatioInfo.GroupRatio)
	costInfo.ActualEstimatedQuota = costInfo.PlannerEstimatedQuota + costInfo.ExecutorEstimatedQuota

	if adjustedUsage := adjustMyCostSavingUsageForBilling(info, usage); adjustedUsage != nil {
		usage = adjustedUsage
	}
	service.PostTextConsumeQuota(c, info, usage, []string{"my_cost_saving"})
	return true, nil
}

func handleMyCostSavingFallback(c *gin.Context, info *relaycommon.RelayInfo, reason string, err *types.NewAPIError) (bool, *types.NewAPIError) {
	if info.CostSaving != nil {
		info.CostSaving.FallbackUsed = true
		info.CostSaving.FallbackReason = reason
	}
	if mycostsaving.GetSettings().FallbackToOriginal {
		logger.LogWarn(c, "my_cost_saving fallback to original model: "+reason)
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return false, types.NewOpenAIError(errors.New(reason), types.ErrorCodeDoRequestFailed, http.StatusInternalServerError)
}

func isMyCostSavingTextRequestSupported(request *dto.GeneralOpenAIRequest) bool {
	if request == nil {
		return false
	}
	if request.Prompt != nil || request.Input != nil {
		return false
	}
	if len(request.Tools) > 0 || len(request.Functions) > 0 || len(request.FunctionCall) > 0 || request.ToolChoice != nil {
		return false
	}
	return len(request.Messages) > 0
}

func appendMyCostSavingAnalysis(messages []dto.Message, analysis string) []dto.Message {
	analysis = strings.TrimSpace(analysis)
	if analysis == "" {
		return messages
	}
	message := dto.Message{
		Role:    "system",
		Content: "Internal analysis for execution. Use it only to solve the user's request; do not mention it to the user.\n" + analysis,
	}
	return append([]dto.Message{message}, messages...)
}

func runMyCostSavingPlanner(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest, plannerModel string, maxTokens int, prompt string) (string, *dto.Usage, *relaycommon.RelayInfo, error) {
	plannerRequest, err := common.DeepCopy(request)
	if err != nil {
		return "", nil, nil, err
	}
	plannerRequest.SetModelName(plannerModel)
	stream := false
	plannerRequest.Stream = &stream
	plannerRequest.StreamOptions = nil
	plannerRequest.Tools = nil
	plannerRequest.ToolChoice = nil
	plannerRequest.Functions = nil
	plannerRequest.FunctionCall = nil
	plannerRequest.ResponseFormat = nil
	if maxTokens > 0 {
		mt := uint(maxTokens)
		plannerRequest.MaxTokens = &mt
		plannerRequest.MaxCompletionTokens = nil
	}
	if strings.TrimSpace(prompt) == "" {
		prompt = mycostsaving.GetSettings().PlannerPrompt
	}
	plannerRequest.Messages = append([]dto.Message{{Role: "system", Content: prompt}}, plannerRequest.Messages...)

	analysis, usage, runInfo, newAPIError := runMyCostSavingNonStream(c, info, plannerRequest, plannerModel, true)
	if newAPIError != nil {
		return "", usage, runInfo, newAPIError
	}
	return strings.TrimSpace(analysis), usage, runInfo, nil
}

func runMyCostSavingExecution(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest, executorModel string) (*dto.Usage, *relaycommon.RelayInfo, *types.NewAPIError) {
	if info.IsStream {
		return runMyCostSavingStream(c, info, request, executorModel)
	}
	_, usage, runInfo, err := runMyCostSavingNonStream(c, info, request, executorModel, false)
	return usage, runInfo, err
}

func runMyCostSavingNonStream(c *gin.Context, baseInfo *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest, modelName string, hidden bool) (string, *dto.Usage, *relaycommon.RelayInfo, *types.NewAPIError) {
	runInfo, adaptor, newAPIError := prepareMyCostSavingAttempt(c, baseInfo, request, modelName, hidden)
	if newAPIError != nil {
		return "", nil, runInfo, newAPIError
	}
	storage, err := requestBodyStorageFromObject(request)
	if err != nil {
		return "", nil, runInfo, types.NewOpenAIError(err, types.ErrorCodeJsonMarshalFailed, http.StatusInternalServerError)
	}
	defer storage.Close()
	body := common.NewReplayableBodyReader(storage)
	resp, err := adaptor.DoRequest(c, runInfo, body)
	if err != nil {
		return "", nil, runInfo, types.NewOpenAIError(err, types.ErrorCodeDoRequestFailed, http.StatusInternalServerError)
	}
	httpResp, ok := resp.(*http.Response)
	if !ok || httpResp == nil {
		return "", nil, runInfo, types.NewOpenAIError(fmt.Errorf("invalid http response"), types.ErrorCodeBadResponse, http.StatusInternalServerError)
	}
	if httpResp.StatusCode != http.StatusOK {
		return "", nil, runInfo, service.RelayErrorHandler(c.Request.Context(), httpResp, false)
	}
	responseBody, readErr := io.ReadAll(httpResp.Body)
	service.CloseResponseBodyGracefully(httpResp)
	if readErr != nil {
		return "", nil, runInfo, types.NewOpenAIError(readErr, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
	}

	var response dto.OpenAITextResponse
	if err := common.Unmarshal(responseBody, &response); err != nil {
		return "", nil, runInfo, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	if oaiError := response.GetOpenAIError(); oaiError != nil && oaiError.Type != "" {
		return "", nil, runInfo, types.WithOpenAIError(*oaiError, httpResp.StatusCode)
	}
	usage := response.Usage
	if usage.PromptTokens == 0 {
		completionTokens := usage.CompletionTokens
		if completionTokens == 0 {
			for _, choice := range response.Choices {
				completionTokens += service.CountTextToken(choice.Message.StringContent()+choice.Message.GetReasoningContent(), runInfo.UpstreamModelName)
			}
		}
		usage = dto.Usage{
			PromptTokens:     runInfo.GetEstimatePromptTokens(),
			CompletionTokens: completionTokens,
			TotalTokens:      runInfo.GetEstimatePromptTokens() + completionTokens,
		}
	}

	if hidden {
		return textFromOpenAIResponse(response), &usage, runInfo, nil
	}
	if baseInfo.CostSaving != nil && baseInfo.CostSaving.HideResponseModel {
		response.Model = baseInfo.CostSaving.OriginalModelName
	}
	response.Usage = *relaycommon.CostSavingVisibleUsage(baseInfo, &usage)
	out, err := common.Marshal(response)
	if err != nil {
		return "", nil, runInfo, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	service.IOCopyBytesGracefully(c, httpResp, out)
	return textFromOpenAIResponse(response), &usage, runInfo, nil
}

func runMyCostSavingStream(c *gin.Context, baseInfo *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest, modelName string) (*dto.Usage, *relaycommon.RelayInfo, *types.NewAPIError) {
	runInfo, adaptor, newAPIError := prepareMyCostSavingAttempt(c, baseInfo, request, modelName, false)
	if newAPIError != nil {
		return nil, runInfo, newAPIError
	}
	storage, err := requestBodyStorageFromObject(request)
	if err != nil {
		return nil, runInfo, types.NewOpenAIError(err, types.ErrorCodeJsonMarshalFailed, http.StatusInternalServerError)
	}
	defer storage.Close()
	body := common.NewReplayableBodyReader(storage)
	resp, err := adaptor.DoRequest(c, runInfo, body)
	if err != nil {
		return nil, runInfo, types.NewOpenAIError(err, types.ErrorCodeDoRequestFailed, http.StatusInternalServerError)
	}
	httpResp, ok := resp.(*http.Response)
	if !ok || httpResp == nil {
		return nil, runInfo, types.NewOpenAIError(fmt.Errorf("invalid http response"), types.ErrorCodeBadResponse, http.StatusInternalServerError)
	}
	if httpResp.StatusCode != http.StatusOK {
		return nil, runInfo, service.RelayErrorHandler(c.Request.Context(), httpResp, false)
	}
	usage, newAPIError := adaptor.DoResponse(c, httpResp, runInfo)
	if newAPIError != nil {
		return nil, runInfo, newAPIError
	}
	if dtoUsage, ok := usage.(*dto.Usage); ok {
		return dtoUsage, runInfo, nil
	}
	return nil, runInfo, nil
}

func prepareMyCostSavingAttempt(c *gin.Context, baseInfo *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest, modelName string, hidden bool) (*relaycommon.RelayInfo, channel.Adaptor, *types.NewAPIError) {
	runInfo := cloneMyCostSavingRelayInfo(baseInfo)
	runInfo.Request = request
	runInfo.OriginModelName = modelName
	runInfo.UpstreamModelName = modelName
	runInfo.IsStream = !hidden && request.IsStream(c.Request)
	runInfo.ShouldIncludeUsage = baseInfo.ShouldIncludeUsage
	runInfo.DisablePing = hidden || baseInfo.DisablePing
	runInfo.CostSaving = baseInfo.CostSaving
	runInfo.InitRequestConversionChain()

	snapshot := snapshotMyCostSavingContext(c)
	defer restoreMyCostSavingContext(c, snapshot)

	channelModel, selectGroup, err := service.CacheGetRandomSatisfiedChannel(&service.RetryParam{
		Ctx:         c,
		TokenGroup:  baseInfo.UsingGroup,
		ModelName:   modelName,
		RequestPath: c.Request.URL.Path,
		Retry:       common.GetPointer(0),
	})
	if err != nil {
		return nil, nil, types.NewOpenAIError(err, types.ErrorCodeGetChannelFailed, http.StatusServiceUnavailable)
	}
	if channelModel == nil {
		return nil, nil, types.NewOpenAIError(fmt.Errorf("no available channel for model %s", modelName), types.ErrorCodeGetChannelFailed, http.StatusServiceUnavailable)
	}
	if selectGroup != "" {
		runInfo.UsingGroup = selectGroup
	}
	fillMyCostSavingChannelMeta(c, runInfo, channelModel, modelName)
	if !isMyCostSavingOpenAICompatible(runInfo.ApiType) {
		return nil, nil, types.NewOpenAIError(fmt.Errorf("channel type %d is not supported by my_cost_saving", channelModel.Type), types.ErrorCodeInvalidApiType, http.StatusBadRequest)
	}

	if err := helper.ModelMappedHelper(c, runInfo, request); err != nil {
		return nil, nil, types.NewOpenAIError(err, types.ErrorCodeChannelModelMappedError, http.StatusBadRequest)
	}
	adaptor := GetAdaptor(runInfo.ApiType)
	if adaptor == nil {
		return nil, nil, types.NewOpenAIError(fmt.Errorf("invalid api type: %d", runInfo.ApiType), types.ErrorCodeInvalidApiType, http.StatusBadRequest)
	}
	adaptor.Init(runInfo)
	convertedRequest, err := adaptor.ConvertOpenAIRequest(c, runInfo, request)
	if err != nil {
		return nil, nil, types.NewOpenAIError(err, types.ErrorCodeConvertRequestFailed, http.StatusBadRequest)
	}
	if convertedOpenAI, ok := convertedRequest.(*dto.GeneralOpenAIRequest); ok {
		*request = *convertedOpenAI
	} else {
		jsonData, err := common.Marshal(convertedRequest)
		if err != nil {
			return nil, nil, types.NewOpenAIError(err, types.ErrorCodeJsonMarshalFailed, http.StatusInternalServerError)
		}
		if err := common.Unmarshal(jsonData, request); err != nil {
			return nil, nil, types.NewOpenAIError(err, types.ErrorCodeConvertRequestFailed, http.StatusBadRequest)
		}
	}
	return runInfo, adaptor, nil
}

func fillMyCostSavingChannelMeta(c *gin.Context, info *relaycommon.RelayInfo, channelModel *model.Channel, modelName string) {
	apiType, _ := common.ChannelType2APIType(channelModel.Type)
	key, index, keyErr := channelModel.GetNextEnabledKey()
	if keyErr != nil {
		common.SysError("my_cost_saving get channel key failed: " + keyErr.Error())
	}
	info.ChannelMeta = &relaycommon.ChannelMeta{
		ChannelType:          channelModel.Type,
		ChannelId:            channelModel.Id,
		ChannelName:          channelModel.Name,
		ChannelIsMultiKey:    channelModel.ChannelInfo.IsMultiKey,
		ChannelMultiKeyIndex: index,
		ChannelBaseUrl:       channelModel.GetBaseURL(),
		ApiType:              apiType,
		ApiKey:               key,
		ChannelCreateTime:    channelModel.CreatedTime,
		ParamOverride:        channelModel.GetParamOverride(),
		HeadersOverride:      channelModel.GetHeaderOverride(),
		ChannelSetting:       channelModel.GetSetting(),
		ChannelOtherSettings: channelModel.GetOtherSettings(),
		UpstreamModelName:    modelName,
	}
	if relaycommon.GetStreamSupportedChannels()[channelModel.Type] {
		info.SupportStreamOptions = true
	}
	if channelModel.OpenAIOrganization != nil {
		info.Organization = *channelModel.OpenAIOrganization
		info.ChannelMeta.Organization = *channelModel.OpenAIOrganization
	}
	if info.ChannelType == constant.ChannelTypeAzure {
		info.ApiVersion = channelModel.Other
		info.ChannelMeta.ApiVersion = channelModel.Other
	}
	if info.ChannelType == constant.ChannelTypeVertexAi {
		info.ApiVersion = channelModel.Other
		info.ChannelMeta.ApiVersion = channelModel.Other
	}
	common.SetContextKey(c, constant.ContextKeyChannelId, channelModel.Id)
	common.SetContextKey(c, constant.ContextKeyChannelName, channelModel.Name)
	common.SetContextKey(c, constant.ContextKeyChannelType, channelModel.Type)
	common.SetContextKey(c, constant.ContextKeyChannelCreateTime, channelModel.CreatedTime)
	common.SetContextKey(c, constant.ContextKeyChannelSetting, info.ChannelSetting)
	common.SetContextKey(c, constant.ContextKeyChannelOtherSetting, info.ChannelOtherSettings)
	common.SetContextKey(c, constant.ContextKeyChannelModelMapping, channelModel.GetModelMapping())
	common.SetContextKey(c, constant.ContextKeyChannelParamOverride, info.ParamOverride)
	common.SetContextKey(c, constant.ContextKeyChannelHeaderOverride, info.HeadersOverride)
	common.SetContextKey(c, constant.ContextKeyChannelOrganization, info.Organization)
	common.SetContextKey(c, constant.ContextKeyChannelAutoBan, channelModel.GetAutoBan())
	common.SetContextKey(c, constant.ContextKeyChannelStatusCodeMapping, channelModel.GetStatusCodeMapping())
	common.SetContextKey(c, constant.ContextKeyChannelIsMultiKey, channelModel.ChannelInfo.IsMultiKey)
	common.SetContextKey(c, constant.ContextKeyChannelMultiKeyIndex, index)
	common.SetContextKey(c, constant.ContextKeyChannelKey, key)
	common.SetContextKey(c, constant.ContextKeyChannelBaseUrl, channelModel.GetBaseURL())
}

func isMyCostSavingOpenAICompatible(apiType int) bool {
	switch apiType {
	case constant.APITypeOpenAI,
		constant.APITypeOpenRouter,
		constant.APITypeXinference,
		constant.APITypeXai,
		constant.APITypeDeepSeek,
		constant.APITypeSiliconFlow,
		constant.APITypeAli,
		constant.APITypeVolcEngine,
		constant.APITypeSubmodel,
		constant.APITypeMiniMax,
		constant.APITypeAdvancedCustom,
		constant.APITypeSub2API,
		constant.APITypeNewAPI,
		constant.APITypeCodex:
		return true
	default:
		return false
	}
}

func applyMyCostSavingAttemptInfo(costInfo *relaycommon.CostSavingInfo, runInfo *relaycommon.RelayInfo, planner bool) {
	if costInfo == nil || runInfo == nil || runInfo.ChannelMeta == nil {
		return
	}
	channelName := runInfo.ChannelMeta.ChannelName
	if planner {
		costInfo.PlannerUpstreamModelName = runInfo.UpstreamModelName
		costInfo.PlannerChannelId = runInfo.ChannelId
		costInfo.PlannerChannelName = channelName
		costInfo.PlannerChannelType = runInfo.ChannelType
		return
	}
	costInfo.ExecutorUpstreamModelName = runInfo.UpstreamModelName
	costInfo.ExecutorChannelId = runInfo.ChannelId
	costInfo.ExecutorChannelName = channelName
	costInfo.ExecutorChannelType = runInfo.ChannelType
	costInfo.ActualUpstreamModelName = runInfo.UpstreamModelName
}

func applyMyCostSavingExecutionInfo(info *relaycommon.RelayInfo, runInfo *relaycommon.RelayInfo) {
	if info == nil || runInfo == nil || runInfo.ChannelMeta == nil {
		return
	}
	info.ChannelMeta = runInfo.ChannelMeta
	info.UsingGroup = runInfo.UsingGroup
	info.UpstreamModelName = runInfo.UpstreamModelName
	info.IsModelMapped = runInfo.IsModelMapped
	info.FinalRequestRelayFormat = runInfo.GetFinalRequestRelayFormat()
	info.RequestConversionChain = append([]types.RelayFormat(nil), runInfo.RequestConversionChain...)
	info.CostSaving = runInfo.CostSaving
}

func BuildTaskCostSavingContext(costInfo *relaycommon.CostSavingInfo) *model.TaskCostSavingContext {
	if costInfo == nil {
		return nil
	}
	actualEstimatedQuota := costInfo.ActualEstimatedQuota
	if costInfo.FallbackUsed {
		actualEstimatedQuota = costInfo.OriginalBilledQuota
	}
	return &model.TaskCostSavingContext{
		Enabled:                   costInfo.Enabled,
		RuleName:                  costInfo.RuleName,
		OriginalModelName:         costInfo.OriginalModelName,
		PlannerModelName:          costInfo.PlannerModelName,
		ExecutorModelName:         costInfo.ExecutorModelName,
		PlannerUpstreamModelName:  costInfo.PlannerUpstreamModelName,
		ExecutorUpstreamModelName: costInfo.ExecutorUpstreamModelName,
		ActualUpstreamModelName:   costInfo.ActualUpstreamModelName,
		PlannerChannelId:          costInfo.PlannerChannelId,
		PlannerChannelName:        costInfo.PlannerChannelName,
		PlannerChannelType:        costInfo.PlannerChannelType,
		ExecutorChannelId:         costInfo.ExecutorChannelId,
		ExecutorChannelName:       costInfo.ExecutorChannelName,
		ExecutorChannelType:       costInfo.ExecutorChannelType,
		AnalysisInjected:          costInfo.AnalysisInjected,
		FallbackUsed:              costInfo.FallbackUsed,
		FallbackReason:            costInfo.FallbackReason,
		HideResponseModel:         costInfo.HideResponseModel,
		OriginalPromptTokens:      costInfo.OriginalPromptTokens,
		PlannerPromptTokens:       costInfo.PlannerPromptTokens,
		PlannerCompletionTokens:   costInfo.PlannerCompletionTokens,
		ExecutorPromptTokens:      costInfo.ExecutorPromptTokens,
		ExecutorCompletionTokens:  costInfo.ExecutorCompletionTokens,
		RawPromptTokens:           costInfo.RawPromptTokens,
		RawCompletionTokens:       costInfo.RawCompletionTokens,
		RawTotalTokens:            costInfo.RawTotalTokens,
		PlannerEstimatedQuota:     costInfo.PlannerEstimatedQuota,
		ExecutorEstimatedQuota:    costInfo.ExecutorEstimatedQuota,
		ActualEstimatedQuota:      actualEstimatedQuota,
		OriginalBilledQuota:       costInfo.OriginalBilledQuota,
		SavingQuota:               maxInt(costInfo.OriginalBilledQuota-actualEstimatedQuota, 0),
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func cloneMyCostSavingRelayInfo(info *relaycommon.RelayInfo) *relaycommon.RelayInfo {
	clone := *info
	clone.ChannelMeta = nil
	clone.Billing = nil
	clone.BillingSource = ""
	clone.SubscriptionId = 0
	clone.SubscriptionPreConsumed = 0
	clone.SubscriptionPostDelta = 0
	clone.TieredBillingSnapshot = nil
	clone.BillingRequestInput = nil
	clone.QuotaClamp = nil
	return &clone
}

func requestBodyStorageFromObject(v any) (common.BodyStorage, error) {
	jsonData, err := common.Marshal(v)
	if err != nil {
		return nil, err
	}
	return common.CreateBodyStorage(jsonData)
}

func textFromOpenAIResponse(response dto.OpenAITextResponse) string {
	var builder strings.Builder
	for _, choice := range response.Choices {
		builder.WriteString(choice.Message.StringContent())
		builder.WriteString(choice.Message.GetReasoningContent())
	}
	return builder.String()
}

func adjustMyCostSavingUsageForBilling(info *relaycommon.RelayInfo, usage *dto.Usage) *dto.Usage {
	return relaycommon.CostSavingBillingUsage(info, usage)
}

func estimateMyCostSavingQuota(modelName string, promptTokens int, completionTokens int, groupRatio float64) int {
	if modelPrice, ok := ratio_setting.GetModelPrice(modelName, false); ok {
		quota, _ := common.QuotaFromFloatChecked(modelPrice * common.QuotaPerUnit * groupRatio)
		return quota
	}
	modelRatio, _, _ := ratio_setting.GetModelRatio(modelName)
	completionRatio := ratio_setting.GetCompletionRatio(modelName)
	quota, _ := common.QuotaFromFloatChecked((float64(promptTokens) + float64(completionTokens)*completionRatio) * modelRatio * groupRatio)
	return quota
}

func snapshotMyCostSavingContext(c *gin.Context) myCostSavingContextSnapshot {
	return myCostSavingContextSnapshot{
		channelId:             common.GetContextKeyInt(c, constant.ContextKeyChannelId),
		channelName:           common.GetContextKeyString(c, constant.ContextKeyChannelName),
		channelType:           common.GetContextKeyInt(c, constant.ContextKeyChannelType),
		channelCreateTime:     c.GetInt64(string(constant.ContextKeyChannelCreateTime)),
		channelSetting:        mustContextValue[dto.ChannelSettings](c, constant.ContextKeyChannelSetting),
		channelOtherSetting:   mustContextValue[dto.ChannelOtherSettings](c, constant.ContextKeyChannelOtherSetting),
		paramOverride:         common.GetContextKeyStringMap(c, constant.ContextKeyChannelParamOverride),
		headerOverride:        common.GetContextKeyStringMap(c, constant.ContextKeyChannelHeaderOverride),
		organization:          common.GetContextKeyString(c, constant.ContextKeyChannelOrganization),
		autoBan:               common.GetContextKeyBool(c, constant.ContextKeyChannelAutoBan),
		modelMapping:          common.GetContextKeyString(c, constant.ContextKeyChannelModelMapping),
		statusCodeMapping:     common.GetContextKeyString(c, constant.ContextKeyChannelStatusCodeMapping),
		channelIsMultiKey:     common.GetContextKeyBool(c, constant.ContextKeyChannelIsMultiKey),
		channelMultiKeyIndex:  common.GetContextKeyInt(c, constant.ContextKeyChannelMultiKeyIndex),
		channelKey:            common.GetContextKeyString(c, constant.ContextKeyChannelKey),
		channelBaseUrl:        common.GetContextKeyString(c, constant.ContextKeyChannelBaseUrl),
		systemPromptOverwrite: common.GetContextKeyBool(c, constant.ContextKeySystemPromptOverride),
		apiVersion:            c.GetString("api_version"),
		region:                c.GetString("region"),
		plugin:                c.GetString("plugin"),
		botId:                 c.GetString("bot_id"),
		useChannel:            c.GetStringSlice("use_channel"),
	}
}

func restoreMyCostSavingContext(c *gin.Context, snapshot myCostSavingContextSnapshot) {
	common.SetContextKey(c, constant.ContextKeyChannelId, snapshot.channelId)
	common.SetContextKey(c, constant.ContextKeyChannelName, snapshot.channelName)
	common.SetContextKey(c, constant.ContextKeyChannelType, snapshot.channelType)
	common.SetContextKey(c, constant.ContextKeyChannelCreateTime, snapshot.channelCreateTime)
	common.SetContextKey(c, constant.ContextKeyChannelSetting, snapshot.channelSetting)
	common.SetContextKey(c, constant.ContextKeyChannelOtherSetting, snapshot.channelOtherSetting)
	common.SetContextKey(c, constant.ContextKeyChannelParamOverride, snapshot.paramOverride)
	common.SetContextKey(c, constant.ContextKeyChannelHeaderOverride, snapshot.headerOverride)
	common.SetContextKey(c, constant.ContextKeyChannelOrganization, snapshot.organization)
	common.SetContextKey(c, constant.ContextKeyChannelAutoBan, snapshot.autoBan)
	common.SetContextKey(c, constant.ContextKeyChannelModelMapping, snapshot.modelMapping)
	common.SetContextKey(c, constant.ContextKeyChannelStatusCodeMapping, snapshot.statusCodeMapping)
	common.SetContextKey(c, constant.ContextKeyChannelIsMultiKey, snapshot.channelIsMultiKey)
	common.SetContextKey(c, constant.ContextKeyChannelMultiKeyIndex, snapshot.channelMultiKeyIndex)
	common.SetContextKey(c, constant.ContextKeyChannelKey, snapshot.channelKey)
	common.SetContextKey(c, constant.ContextKeyChannelBaseUrl, snapshot.channelBaseUrl)
	common.SetContextKey(c, constant.ContextKeySystemPromptOverride, snapshot.systemPromptOverwrite)
	c.Set("api_version", snapshot.apiVersion)
	c.Set("region", snapshot.region)
	c.Set("plugin", snapshot.plugin)
	c.Set("bot_id", snapshot.botId)
	c.Set("use_channel", snapshot.useChannel)
}

func mustContextValue[T any](c *gin.Context, key constant.ContextKey) T {
	value, _ := common.GetContextKeyType[T](c, key)
	return value
}
