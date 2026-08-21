package controller

import (
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

type myCostSavingModelOption struct {
	Model               string   `json:"model"`
	SupportedGroups     []string `json:"supported_groups"`
	SupportedGroupCount int      `json:"supported_group_count"`
	ChannelCount        int      `json:"channel_count"`
	AllGroupsSupported  bool     `json:"all_groups_supported"`
}

// GetMyCostSavingModels returns active model availability for the admin rule
// editor. Models are grouped by active channel abilities, not just model
// metadata, so unsupported channels never become selectable by accident.
func GetMyCostSavingModels(c *gin.Context) {
	groups := make([]string, 0)
	for _, value := range strings.Split(c.Query("groups"), ",") {
		value = strings.TrimSpace(value)
		if value != "" && !containsString(groups, value) {
			groups = append(groups, value)
		}
	}

	availability, err := model.GetEnabledModelsByGroups(groups)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	type modelSummary struct {
		supportedGroups []string
		channelCount    int
	}
	summaries := make(map[string]*modelSummary)
	for _, item := range availability {
		summary, ok := summaries[item.Model]
		if !ok {
			summary = &modelSummary{}
			summaries[item.Model] = summary
		}
		if !containsString(summary.supportedGroups, item.Group) {
			summary.supportedGroups = append(summary.supportedGroups, item.Group)
		}
		summary.channelCount += item.ChannelCount
	}

	modelNames := make([]string, 0, len(summaries))
	for modelName := range summaries {
		modelNames = append(modelNames, modelName)
	}
	sort.Strings(modelNames)

	models := make([]myCostSavingModelOption, 0, len(modelNames))
	for _, modelName := range modelNames {
		summary := summaries[modelName]
		sort.Strings(summary.supportedGroups)
		allGroupsSupported := len(groups) == 0 || len(summary.supportedGroups) == len(groups)
		models = append(models, myCostSavingModelOption{
			Model:               modelName,
			SupportedGroups:     summary.supportedGroups,
			SupportedGroupCount: len(summary.supportedGroups),
			ChannelCount:        summary.channelCount,
			AllGroupsSupported:  allGroupsSupported,
		})
	}

	common.ApiSuccess(c, gin.H{
		"selected_groups": groups,
		"models":          models,
	})
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
