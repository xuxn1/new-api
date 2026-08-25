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
