package model

import "github.com/QuantumNous/new-api/common"

// GroupModelAvailability describes the active channels that can serve a model
// in a group. It is used by the admin-only my-cost saving editor.
type GroupModelAvailability struct {
	Group        string `json:"group"`
	Model        string `json:"model"`
	ChannelCount int    `json:"channel_count"`
}

// GetEnabledModelsByGroups returns models from enabled abilities on enabled
// channels. An empty groups slice means all groups.
func GetEnabledModelsByGroups(groups []string) ([]GroupModelAvailability, error) {
	type row struct {
		GroupName    string
		Model        string
		ChannelCount int
	}

	query := DB.Table("abilities").
		Select("abilities."+commonGroupCol+" as group_name, abilities.model as model, COUNT(DISTINCT abilities.channel_id) as channel_count").
		Joins("JOIN channels ON channels.id = abilities.channel_id").
		Where("abilities.enabled = ? AND channels.status = ?", true, common.ChannelStatusEnabled)
	if len(groups) > 0 {
		query = query.Where("abilities."+commonGroupCol+" IN ?", groups)
	}

	var rows []row
	if err := query.
		Group("abilities." + commonGroupCol + ", abilities.model").
		Order("abilities." + commonGroupCol).
		Order("abilities.model").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	result := make([]GroupModelAvailability, 0, len(rows))
	for _, item := range rows {
		result = append(result, GroupModelAvailability{
			Group:        item.GroupName,
			Model:        item.Model,
			ChannelCount: item.ChannelCount,
		})
	}
	return result, nil
}
