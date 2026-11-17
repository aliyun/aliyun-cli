package safety

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPolicyEmptyRulesAndFilesystemErrors(t *testing.T) {
	t.Setenv(EnvSafetyPolicyEnabled, "true")
	t.Setenv(EnvSafetyPolicyRules, ", =deny,*:Delete*=,bad=invalid,ecs:List*=allow,")
	policy := MergePolicyFromEnv(nil)
	require.True(t, policy.Enabled)
	require.Equal(t, []Rule{{Pattern: "ecs:List*", Action: ActionAllow}}, policy.Rules)
	require.Equal(t, "ecs:List*=allow", serializeRulesForEnv(append(policy.Rules, Rule{}, Rule{Pattern: "empty"})))
	result := (&Policy{Enabled: true, Rules: []Rule{{Pattern: "*"}}}).Check(CommandInfo{Product: "ecs", ApiOrMethod: "DeleteInstance"})
	require.Equal(t, ActionDeny, result.Action)
	require.False(t, matchPattern("", "ecs:DeleteInstance"))
	dir := t.TempDir()
	require.NoError(t, SavePolicy(dir, nil))
	loaded, err := LoadPolicy(dir)
	require.NoError(t, err)
	require.Equal(t, DefaultPolicy(), loaded)
	require.NoError(t, os.WriteFile(GetPolicyFilePath(dir), []byte(`{"enabled":false}`), 0600))
	loaded, err = LoadPolicy(dir)
	require.NoError(t, err)
	require.NotNil(t, loaded.Rules)
	require.Empty(t, loaded.Rules)
	bad := t.TempDir()
	require.NoError(t, os.Mkdir(GetPolicyFilePath(bad), 0700))
	_, err = LoadEffectivePolicy(bad)
	require.Error(t, err)
	envs := map[string]string{}
	MergeSafetyPolicyPathIntoEnvs(bad, envs)
	require.Equal(t, GetPolicyFilePath(bad), envs[EnvSafetyPolicyFile])
	_, exists := envs[EnvSafetyPolicyEnabled]
	require.False(t, exists)
	blocked := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(blocked, []byte("blocked"), 0600))
	require.Error(t, SavePolicy(filepath.Join(blocked, "child"), nil))
}
