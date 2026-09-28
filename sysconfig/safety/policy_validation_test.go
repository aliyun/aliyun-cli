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

func TestStrictEnvJSONRules(t *testing.T) {
	for _, tc := range []struct {
		name, raw, serialized string
		want                  []Rule
	}{
		{"empty", `{"rules":[]}`, "", []Rule{}},
		{"normalized", ` {"rules":[{"pattern":" esa:* ","action":" DeNy "}]} `, "esa:*=deny", []Rule{{"esa:*", ActionDeny}}},
		{"policy wrapper", `{"enabled":false,"rules":[{"pattern":"esa:list-sites","action":"allow"},{"pattern":"*","action":"deny"}]}`, "esa:list-sites=allow,*=deny", []Rule{{"esa:list-sites", ActionAllow}, {"*", ActionDeny}}},
		{"confirmation", `{"rules":[{"pattern":"ecs:*","action":"confirm"},{"pattern":"oss:*","action":"forbid"}]}`, "ecs:*=confirm,oss:*=forbid", []Rule{{"ecs:*", ActionConfirm}, {"oss:*", ActionForbid}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(EnvSafetyPolicyEnabled, "true")
			t.Setenv(EnvSafetyPolicyRules, tc.raw)
			dir := t.TempDir()
			require.NoError(t, SavePolicy(dir, &Policy{Rules: []Rule{{"old:*", ActionDeny}}}))
			got, err := LoadEffectivePolicy(dir)
			require.NoError(t, err)
			require.True(t, got.Enabled)
			require.Equal(t, tc.want, got.Rules)

			// Child plugins still receive the existing pattern=action format.
			envs := map[string]string{}
			MergeSafetyPolicyPathIntoEnvs(dir, envs)
			require.Equal(t, "true", envs[EnvSafetyPolicyEnabled])
			require.Equal(t, tc.serialized, envs[EnvSafetyPolicyRules])
			t.Setenv(EnvSafetyPolicyRules, envs[EnvSafetyPolicyRules])
			roundTrip, err := LoadEffectivePolicy(dir)
			require.NoError(t, err)
			require.Equal(t, got, roundTrip)
		})
	}

	t.Run("rules do not enable the policy", func(t *testing.T) {
		t.Setenv(EnvSafetyPolicyEnabled, "false")
		t.Setenv(EnvSafetyPolicyRules, `{"enabled":true,"rules":[{"pattern":"*","action":"deny"}]}`)
		got, err := LoadEffectivePolicy(t.TempDir())
		require.NoError(t, err)
		require.False(t, got.Enabled)
		require.Equal(t, []Rule{{"*", ActionDeny}}, got.Rules)
	})
}

func TestStrictEnvJSONRulesRejectInvalidInput(t *testing.T) {
	for _, raw := range []string{
		`{"rules":`,
		`{}`,
		`{"rules":null}`,
		`{"rules":{}}`,
		`{"rules":[],"extra":true}`,
		`{"enabled":"true","rules":[]}`,
		`{"rules":[{"pattern":"*","action":"deny","extra":true}]}`,
		`{"rules":[{"pattern":" ","action":"deny"}]}`,
		`{"rules":[{"pattern":"*"}]}`,
		`{"rules":[{"pattern":"*","action":"dney"},{"pattern":"*","action":"allow"}]}`,
		`{"rules":[null]}`,
		`{"rules":[]} {"rules":[]}`,
		`{"rules":[]} garbage`,
	} {
		t.Run(raw, func(t *testing.T) {
			t.Setenv(EnvSafetyPolicyEnabled, "true")
			t.Setenv(EnvSafetyPolicyRules, raw)
			got, err := LoadEffectivePolicy(t.TempDir())
			require.ErrorContains(t, err, EnvSafetyPolicyRules)
			require.Nil(t, got)
		})
	}
}
