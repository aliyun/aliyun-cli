package mcpproxy

import (
	"errors"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/stretchr/testify/require"
)

func TestMCPAuthenticationErrorData(t *testing.T) {
	err := newMCPAuthenticationError("profile configuration failed", "check_configuration", "Check the selected profile.")
	require.EqualError(t, err, "profile configuration failed")
	var structured cli.StructuredError
	require.False(t, errors.As(err, &structured))
	var agentErr *cli.AgentError
	require.False(t, errors.As(err, &agentErr))
	require.False(t, cli.IsAIRecoveryEligible(err))
	var authenticationErr *AuthenticationError
	require.ErrorAs(t, err, &authenticationErr)
	require.Equal(t, "profile configuration failed", authenticationErr.Message)
	require.Equal(t, "check_configuration", authenticationErr.Action)
	require.Equal(t, "Check the selected profile.", authenticationErr.Hint)
}
