package controller

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	taskdto "github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestShouldRetryAfterAffinityUpstreamThrottle(t *testing.T) {
	gin.SetMode(gin.TestMode)

	setting := operation_setting.GetChannelAffinitySetting()
	originalSetting := *setting
	t.Cleanup(func() {
		*setting = originalSetting
	})
	setting.Enabled = true
	setting.DefaultTTLSeconds = 60
	setting.Rules = []operation_setting.ChannelAffinityRule{
		{
			Name:       "test affinity retry",
			ModelRegex: []string{"^gpt-test$"},
			KeySources: []operation_setting.ChannelAffinityKeySource{
				{Type: "request_header", Key: "X-Affinity-Key"},
			},
			SkipRetryOnFailure: true,
			IncludeRuleName:    true,
			IncludeUsingGroup:  true,
		},
	}

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	ctx.Request.Header.Set("X-Affinity-Key", "tenant-a")

	_, found := service.GetPreferredChannelByAffinity(ctx, "gpt-test", "default")
	require.False(t, found)
	service.MarkChannelAffinityUsed(ctx, "default", 1001)
	require.True(t, service.ShouldSkipRetryAfterChannelAffinityFailure(ctx))

	channelErr := types.NewError(errors.New("no enabled keys"), types.ErrorCodeChannelNoAvailableKey)
	require.False(t, shouldRetry(ctx, channelErr, 1))

	err := types.NewOpenAIError(errors.New("upstream overloaded"), types.ErrorCodeBadResponseStatusCode, http.StatusTooManyRequests)
	processChannelError(ctx, *types.NewChannelError(1001, 1, "affinity-channel", false, "", false), err)

	require.False(t, service.ShouldSkipRetryAfterChannelAffinityFailure(ctx))
	require.True(t, shouldRetry(ctx, err, 1))
}

func TestShouldRetryTaskRelayAfterAffinityUpstreamThrottle(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Set("channel_affinity_skip_retry_on_failure", true)

	taskErr := &taskdto.TaskError{
		Code:       "rate_limited",
		Message:    "upstream overloaded",
		StatusCode: http.StatusTooManyRequests,
		LocalError: false,
		Error:      errors.New("upstream overloaded"),
	}

	require.True(t, shouldRetryTaskRelay(ctx, 1001, taskErr, 1))
}
