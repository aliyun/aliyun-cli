package agentbay

import (
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/endpoint"
	"github.com/stretchr/testify/require"
)

func TestAgentBayRejectsInvalidEndpointSuffixBeforeLoadingProfile(t *testing.T) {
	t.Setenv(endpoint.EnvSuffix, "invalid.example")
	envs, err := NewContext(nil).BuildExecEnv(agentBayCLIOptions{})
	require.ErrorContains(t, err, endpoint.EnvSuffix)
	require.Nil(t, envs)
}
