package cli

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAgentErrorVersionPreservesOriginalEnvelope(t *testing.T) {
	cause := errors.New("failed")
	original := NewAgentError(AgentErrorEnvelope{Message: "failed", Recovery: AgentErrorRecovery{Action: "inspect", Hint: "check state"}}, cause)
	versioned := original.WithSchemaVersion()
	require.Empty(t, original.Envelope().SchemaVersion)
	require.Equal(t, AgentErrorSchemaVersion, versioned.Envelope().SchemaVersion)
	require.Equal(t, original.Error(), versioned.Error())
	require.ErrorIs(t, versioned, cause)
}
