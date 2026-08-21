package model

type ChannelUsageStat struct {
	ChannelID      int   `gorm:"column:channel_id"`
	TodayUsedQuota int64 `gorm:"column:today_used_quota"`
	LastCallTime   int64 `gorm:"column:last_call_time"`
}

// GetChannelUsageStatsByIDs returns today's used quota and the latest call
// timestamp for the provided channel IDs. It is used by the admin channel list
// so the page can surface daily activity without mutating channel state.
func GetChannelUsageStatsByIDs(channelIDs []int, todayStart int64) (map[int]ChannelUsageStat, error) {
	stats := make(map[int]ChannelUsageStat)
	if len(channelIDs) == 0 || LOG_DB == nil {
		return stats, nil
	}

	type row struct {
		ChannelID      int   `gorm:"column:channel_id"`
		TodayUsedQuota int64 `gorm:"column:today_used_quota"`
		LastCallTime   int64 `gorm:"column:last_call_time"`
	}

	var rows []row
	err := LOG_DB.Table("logs").
		Select(
			"channel_id, "+
				"COALESCE(SUM(CASE WHEN type = ? AND created_at >= ? THEN quota ELSE 0 END), 0) AS today_used_quota, "+
				"COALESCE(MAX(CASE WHEN type IN (?, ?) THEN created_at ELSE 0 END), 0) AS last_call_time",
			LogTypeConsume,
			todayStart,
			LogTypeConsume,
			LogTypeError,
		).
		Where("channel_id IN ?", channelIDs).
		Where("type IN ?", []int{LogTypeConsume, LogTypeError}).
		Group("channel_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		stats[row.ChannelID] = ChannelUsageStat{
			ChannelID:      row.ChannelID,
			TodayUsedQuota: row.TodayUsedQuota,
			LastCallTime:   row.LastCallTime,
		}
	}

	return stats, nil
}
