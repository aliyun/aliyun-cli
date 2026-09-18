package safety

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStrictPolicyValidationBoundaries(t *testing.T) {
	require.ErrorContains(t, validatePolicy(nil), "nil")
	for _, tc := range []struct{ raw, message string }{
		{`{"enabled":true} {}`, "multiple JSON values"},
		{`{"enabled":true} garbage`, "trailing data"},
		{`{"rules":[{"pattern":"  ","action":"deny"}]}`, "pattern is empty"},
		{`{"rules":[{"pattern":"*","action":"unknown"}]}`, "action"},
	} {
		t.Run(tc.message, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.WriteFile(GetPolicyFilePath(dir), []byte(tc.raw), 0600))
			p, err := LoadPolicy(dir)
			require.ErrorContains(t, err, tc.message)
			require.Nil(t, p)
		})
	}
	for _, tc := range []struct{ raw, message string }{{"x=deny,", "entry 2 is empty"}, {"x", "pattern=action"}, {"=deny", "empty pattern"}, {"x=bogus", "invalid action"}} {
		t.Run(tc.raw, func(t *testing.T) {
			t.Setenv(EnvSafetyPolicyEnabled, "true")
			t.Setenv(EnvSafetyPolicyRules, tc.raw)
			p, err := MergePolicyFromEnvStrict(nil)
			require.Nil(t, p)
			require.ErrorContains(t, err, tc.message)
		})
	}
	t.Run("normalized rules and independent copy", func(t *testing.T) {
		t.Setenv(EnvSafetyPolicyEnabled, " true ")
		t.Setenv(EnvSafetyPolicyRules, " ecs:* = DeNy , oss:* = FORBID ")
		p, err := MergePolicyFromEnvStrict(nil)
		require.NoError(t, err)
		require.True(t, p.Enabled)
		require.Equal(t, []Rule{{"ecs:*", ActionDeny}, {"oss:*", ActionForbid}}, p.Rules)
		require.Equal(t, ActionConfirm, p.Check(CommandInfo{Product: "oss", ApiOrMethod: "rm"}).Action)
	})
	t.Run("empty override clears rules", func(t *testing.T) {
		t.Setenv(EnvSafetyPolicyEnabled, "false")
		t.Setenv(EnvSafetyPolicyRules, " ")
		p, err := MergePolicyFromEnvStrict(&Policy{Enabled: true, Rules: []Rule{{"*", ActionDeny}}})
		require.NoError(t, err)
		require.False(t, p.Enabled)
		require.Empty(t, p.Rules)
	})
	t.Run("invalid base cannot be overridden", func(t *testing.T) {
		t.Setenv(EnvSafetyPolicyRules, "*=allow")
		_, err := MergePolicyFromEnvStrict(&Policy{Rules: []Rule{{"", ActionDeny}}})
		require.ErrorContains(t, err, "pattern is empty")
	})
}

func TestStrictPolicyRejectsUnknownInMemoryActionAndBoolean(t *testing.T) {
	p := &Policy{Enabled: true, Rules: []Rule{{"*", Action("typo")}}}
	require.Equal(t, ActionDeny, p.Check(CommandInfo{Product: "oss", ApiOrMethod: "rm"}).Action)
	t.Setenv(EnvSafetyPolicyEnabled, "not-a-boolean")
	_, err := MergePolicyFromEnvStrict(nil)
	require.ErrorContains(t, err, "invalid boolean")
}
