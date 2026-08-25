package service

import (
	"errors"
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/stretchr/testify/require"
)

func TestIsUpstreamQuotaExhaustedError(t *testing.T) {
	err := types.NewOpenAIError(
		errors.New("Insufficient account balance"),
		types.ErrorCodeBadResponseStatusCode,
		http.StatusForbidden,
	)

	require.True(t, IsUpstreamQuotaExhaustedError(err))
}

func TestIsUpstreamQuotaExhaustedErrorIgnoresLocalUserQuota(t *testing.T) {
	err := types.NewErrorWithStatusCode(
		errors.New("余额不足"),
		types.ErrorCodeInsufficientUserQuota,
		http.StatusForbidden,
	)

	require.False(t, IsUpstreamQuotaExhaustedError(err))
}

func TestIsUpstreamRateLimitedError(t *testing.T) {
	err := types.NewOpenAIError(
		errors.New("Upstream rate limit exceeded, please retry later"),
		types.ErrorCodeBadResponseStatusCode,
		http.StatusTooManyRequests,
	)

	require.True(t, IsUpstreamRateLimitedError(err))
}

func TestIsUpstreamRateLimitedErrorIgnoresLocalUserQuota(t *testing.T) {
	err := types.NewErrorWithStatusCode(
		errors.New("user quota limited"),
		types.ErrorCodeInsufficientUserQuota,
		http.StatusTooManyRequests,
	)

	require.False(t, IsUpstreamRateLimitedError(err))
}
