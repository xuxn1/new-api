/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type myCostSavingModelsResponse struct {
	Success bool `json:"success"`
	Data    struct {
		SelectedGroups []string         `json:"selected_groups"`
		Models         []map[string]any `json:"models"`
	} `json:"data"`
}

func TestGetMyCostSavingModelsFiltersPartialGroupSupport(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.Create(&[]model.Channel{
		{Id: 1, Type: 1, Key: "key-1", Status: common.ChannelStatusEnabled, Name: "default-channel", Group: "default", Models: "gpt-5,gpt-5.4-mini"},
		{Id: 2, Type: 1, Key: "key-2", Status: common.ChannelStatusEnabled, Name: "default-channel-2", Group: "default", Models: "gpt-5"},
		{Id: 3, Type: 1, Key: "key-3", Status: common.ChannelStatusEnabled, Name: "vip-channel", Group: "vip", Models: "gpt-5"},
	}).Error)
	require.NoError(t, db.Create(&[]model.Ability{
		{Group: "default", Model: "gpt-5", ChannelId: 1, Enabled: true},
		{Group: "default", Model: "gpt-5", ChannelId: 2, Enabled: true},
		{Group: "vip", Model: "gpt-5", ChannelId: 3, Enabled: true},
		{Group: "default", Model: "gpt-5.4-mini", ChannelId: 1, Enabled: true},
		{Group: "vip", Model: "gpt-5.4-mini", ChannelId: 3, Enabled: false},
	}).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/my-cost-saving/models?groups=default,vip", nil)

	GetMyCostSavingModels(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	var payload myCostSavingModelsResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))
	require.True(t, payload.Success)
	require.Equal(t, []string{"default", "vip"}, payload.Data.SelectedGroups)
	require.Len(t, payload.Data.Models, 2)

	first := payload.Data.Models[0]
	second := payload.Data.Models[1]
	if first["model"] == "gpt-5.4-mini" {
		first, second = second, first
	}

	assert.Equal(t, "gpt-5", first["model"])
	assert.Equal(t, true, first["all_groups_supported"])
	assert.Equal(t, []any{"default", "vip"}, first["supported_groups"])
	assert.Equal(t, float64(3), first["channel_count"])

	assert.Equal(t, "gpt-5.4-mini", second["model"])
	assert.Equal(t, false, second["all_groups_supported"])
	assert.Equal(t, []any{"default"}, second["supported_groups"])
	assert.Equal(t, float64(1), second["channel_count"])
}
