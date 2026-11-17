package telemetry

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSensitiveToolParameterKeywords(t *testing.T) {
	t.Setenv("ALIBABA_CLOUD_BEARER_TOKEN_HEADER_KEY", "")
	for _, name := range []string{
		"--sts-token", "--access-key-id", "--access-key-secret", "--password",
		"--AccessKeyID", "--ACCESS_KEY_SECRET", "--db.password", "--database-password",
		"--client-secret", "--session-token", "--security-token", "--bearer-token",
		"--oidc-token-value", "--oidc-token-file", "--bearer-token-header-key",
		"--private-key", "--api_key", "--authorization", "--cookie", "--credentials-file",
		"--passwd", "--passphrase", "--pass", "--AK", "--akid", "--aksk", "--sk", "--pwd", "--accessid", "--accesssecret",
		"--auth", "--bearer", "--x-acs-signature", "--X_ACS_SIGNATURE", "--X.Acs.Signature", "--XAcsSignature", "--signature",
		"--db.auth", "--headers.bearer",
	} {
		t.Run(name, func(t *testing.T) { require.True(t, sensitiveToolParameter(name)) })
	}
	for _, name := range []string{"--name", "--tag", "--region", "--key", "--author", "--user", "-p", "-t", "password", ""} {
		t.Run("ordinary_"+name, func(t *testing.T) { require.False(t, sensitiveToolParameter(name)) })
	}
}

func TestSensitiveToolParameterReusesRuntimeHeaderPolicy(t *testing.T) {
	const headerEnv = "ALIBABA_CLOUD_BEARER_TOKEN_HEADER_KEY"
	t.Setenv(headerEnv, "")
	require.False(t, sensitiveToolParameter("--x-custom-identity"))
	t.Setenv(headerEnv, "X-Custom-Identity")
	for _, name := range []string{"--x-custom-identity", "--X-Custom-Identity", "--X_Custom_Identity"} {
		require.True(t, sensitiveToolParameter(name))
		require.Equal(t, []string{name, "--verbose"}, inferToolParameterNames([]string{name, "--customer-sensitive-value", "--verbose"}))
	}
	t.Setenv(headerEnv, "")
	require.False(t, sensitiveToolParameter("--x-custom-identity"), "custom runtime header names must not persist across invocations")
}

func TestInferToolParameterNamesSkipsSensitiveValues(t *testing.T) {
	t.Setenv("ALIBABA_CLOUD_BEARER_TOKEN_HEADER_KEY", "")
	const sensitive = "--customer-secret-token"
	for _, test := range []struct {
		name string
		args []string
		want []string
	}{
		{"known host token", []string{"--sts-token", sensitive, "--verbose"}, []string{"--sts-token", "--verbose"}},
		{"access key ID", []string{"--access-key-id", sensitive, "--verbose"}, []string{"--access-key-id", "--verbose"}},
		{"access key secret", []string{"--access-key-secret", sensitive, "--verbose"}, []string{"--access-key-secret", "--verbose"}},
		{"password", []string{"--password", sensitive, "--verbose"}, []string{"--password", "--verbose"}},
		{"password alias", []string{"--pass", sensitive, "--verbose"}, []string{"--pass", "--verbose"}},
		{"runtime auth", []string{"--auth", sensitive, "--verbose"}, []string{"--auth", "--verbose"}},
		{"runtime bearer", []string{"--bearer", sensitive, "--verbose"}, []string{"--bearer", "--verbose"}},
		{"runtime signature header", []string{"--x-acs-signature", sensitive, "--verbose"}, []string{"--x-acs-signature", "--verbose"}},
		{"runtime signature separator variant", []string{"--X_ACS_SIGNATURE", sensitive, "--verbose"}, []string{"--X_ACS_SIGNATURE", "--verbose"}},
		{"signature camel case", []string{"--XAcsSignature", sensitive, "--verbose"}, []string{"--XAcsSignature", "--verbose"}},
		{"runtime dotted auth", []string{"--db.auth", sensitive, "--verbose"}, []string{"--db.auth", "--verbose"}},
		{"registered looking value", []string{"--password", "--region", "--verbose"}, []string{"--password", "--verbose"}},
		{"value resembling sensitive name", []string{"--password", "--sts-token", "--verbose"}, []string{"--password", "--verbose"}},
		{"aliases and repeats", []string{"--DB_Password", sensitive, "--DB_Password", sensitive, "--AccessKeyID", sensitive}, []string{"--DB_Password", "--DB_Password", "--AccessKeyID"}},
		{"inline equals", []string{"--password=" + sensitive, "--verbose"}, []string{"--password", "--verbose"}},
		{"inline colon", []string{"--password:" + sensitive, "--verbose"}, []string{"--password", "--verbose"}},
		{"empty inline equals is conservative", []string{"--password=", sensitive, "--verbose"}, []string{"--password", "--verbose"}},
		{"empty inline colon is conservative", []string{"--password:", sensitive, "--verbose"}, []string{"--password", "--verbose"}},
		{"delimiter after sensitive name", []string{"--password", "--", sensitive, "--verbose"}, []string{"--password"}},
		{"delimiter after empty inline", []string{"--password=", "--", sensitive, "--verbose"}, []string{"--password"}},
		{"delimiter before sensitive name", []string{"--verbose", "--", "--password", sensitive}, []string{"--verbose"}},
		{"missing value", []string{"--password"}, []string{"--password"}},
		{"ordinary heuristic preserved", []string{"--name", "--ambiguous-value", "--tag", "one", "two"}, []string{"--name", "--ambiguous-value", "--tag"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			original := append([]string(nil), test.args...)
			names := inferToolParameterNames(test.args)
			require.Equal(t, test.want, names)
			require.Equal(t, original, test.args, "observing names must not mutate execution argv")
			for _, commandType := range []string{CommandTypePluginGo, CommandTypeCLIExt} {
				execution := &activeCommandExecution{scope: Scope{CommandType: commandType}, rootArgs: append([]string{"demo", "list"}, test.args...)}
				event, err := BuildEvent(BuildInput{Scope: execution.scope, RootArgs: execution.rootArgs, ParameterNames: executionParameterNames(execution)})
				require.NoError(t, err)
				require.Equal(t, test.want, event.Params)
				require.Equal(t, test.want, newUploadEvent(event).Attributes["parameterNames"])
				data, err := json.Marshal(newUploadEvent(event))
				require.NoError(t, err)
				require.NotContains(t, string(data), sensitive)
			}
		})
	}
}
