package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/cliext"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/safety"
	"github.com/stretchr/testify/require"
)

func extensionPolicyHome(t *testing.T, pattern string, action safety.Action) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOMEDRIVE", "")
	t.Setenv("HOMEPATH", "")
	t.Setenv("ALIBABA_CLOUD_SAFETY_SKIP_CONFIRM", "")
	for _, name := range []string{safety.EnvSafetyPolicyEnabled, safety.EnvSafetyPolicyRules} {
		t.Setenv(name, "")
		require.NoError(t, os.Unsetenv(name))
	}
	previousLanguage := i18n.GetLanguage()
	i18n.SetLanguage("en")
	t.Cleanup(func() { i18n.SetLanguage(previousLanguage) })
	dir := filepath.Join(home, ".aliyun")
	require.NoError(t, os.MkdirAll(dir, 0700))
	require.NoError(t, safety.SavePolicy(dir, &safety.Policy{Enabled: true, Rules: []safety.Rule{{Pattern: pattern, Action: action}}}))
	return dir
}

func extensionRootContext() (*cli.Command, *cli.Context) {
	root := newRootCommand(config.NewProfile("default"), io.Discard)
	ctx := cli.NewCommandContext(io.Discard, io.Discard)
	ctx.EnterCommand(root)
	return root, ctx
}

func TestEveryExtensionDeniedBeforeDispatch(t *testing.T) {
	extensionPolicyHome(t, "*restore*", safety.ActionDeny)
	// Explicit inventory includes hidden extensions and providers with nested
	// commands or DisablePersistentFlags. All go through the real root router.
	names := []string{"ossutil", "agentbay", "otsutil", "spark-submit", "kmscli", "lindorm", "mseutil", "acrutil", "codeup-cli", "saectl", "appmanager", "computenest-cli", "ecctl", "esa-cli", "flow-cli", "cms2", "maxc", "iact3", "rostran"}
	cli.DisableExitCode()
	t.Cleanup(cli.EnableExitCode)
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			root, _ := extensionRootContext()
			called := false
			root.GetSubCommand(name).Run = func(*cli.Context, []string) error { called = true; return nil }
			var stderr bytes.Buffer
			ctx := cli.NewCommandContext(io.Discard, &stderr)
			ctx.EnterCommand(root)
			root.Execute(ctx, []string{name, "restore"})
			require.False(t, called, "extension must never start")
			require.Contains(t, stderr.String(), "blocked by safety policy")
		})
	}
}

func TestExtensionSafetyRoutingAndMatching(t *testing.T) {
	for _, tc := range []struct {
		name, pattern string
		args          []string
		blocked       bool
	}{
		{"exact", "saectl:restore", []string{"saectl", "restore"}, true},
		{"case insensitive", "*:RESTORE", []string{"saectl", "restore"}, true},
		{"nested acr", "acrutil:diagnosis:restore", []string{"acrutil", "diagnosis", "restore"}, true},
		{"global flag before extension", "saectl:restore", []string{"--profile", "test", "saectl", "restore"}, true},
		{"colon global flag", "saectl:restore", []string{"--profile:test", "saectl", "restore"}, true},
		{"unknown global flag", "saectl:restore", []string{"--custom=value", "saectl", "restore"}, true},
		{"config flag after extension", "saectl:restore", []string{"saectl", "--profile", "test", "restore"}, true},
		{"deny ignores yes", "*restore*", []string{"saectl", "restore", "--yes"}, true},
		{"deny ignores short yes", "*restore*", []string{"saectl", "restore", "-y"}, true},
		{"global yes cannot override deny", "*restore*", []string{"--yes", "saectl", "restore"}, true},
		{"literal help value", "*restore*", []string{"saectl", "restore", "--name", "--help"}, true},
		{"help after separator", "*restore*", []string{"saectl", "restore", "--", "--help"}, true},
		{"help flag with operation", "*restore*", []string{"saectl", "restore", "--help"}, true},
		{"provider version boolean", "*restore*", []string{"saectl", "--version", "restore"}, true},
		{"unknown boolean", "*restore*", []string{"saectl", "--force", "restore"}, true},
		{"unmatched", "*restore*", []string{"saectl", "status"}, false},
		{"core unaffected", "*", []string{"configure", "list"}, false},
		{"oss owns its guard", "*", []string{"oss", "rm", "oss://test/key"}, false},
		{"plugin owns its guard", "*", []string{"fc", "delete-function"}, false},
		{"empty invocation", "*", []string{"saectl"}, false},
		{"help", "*", []string{"saectl", "help"}, false},
		{"help flag", "*", []string{"saectl", "--help"}, false},
		{"short help", "*", []string{"saectl", "-h"}, false},
		{"version", "*", []string{"saectl", "version"}, false},
		{"version flag", "*", []string{"saectl", "--version"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			extensionPolicyHome(t, tc.pattern, safety.ActionDeny)
			_, ctx := extensionRootContext()
			original := append([]string(nil), tc.args...)
			guard := &cli.Command{}
			cliext.AttachSafetyPolicy(guard, map[string]bool{"saectl": true, "acrutil": true})
			_, err := guard.BeforeParseRoute(ctx, tc.args)
			if tc.blocked {
				require.ErrorContains(t, err, "blocked by safety policy")
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, original, tc.args, "must preserve provider argv")
			require.False(t, ctx.Flags().Get("yes").IsAssigned(), "must not mutate parser state")
		})
	}
}

func TestExtensionSafetyConfigPath(t *testing.T) {
	extensionPolicyHome(t, "*", safety.ActionAllow)
	dir := t.TempDir()
	require.NoError(t, safety.SavePolicy(dir, &safety.Policy{Enabled: true, Rules: []safety.Rule{{Pattern: "*restore*", Action: safety.ActionDeny}}}))
	path := filepath.Join(dir, "config.json")
	for _, args := range [][]string{
		{"--config-path", path, "saectl", "restore"},
		{"saectl", "restore", "--config-path", path},
		{"saectl", "--config-path=" + path, "restore"},
		{"saectl", "--config-path:" + path, "restore"},
		{"codeup-cli", "restore", "--config-path", path},
	} {
		root, ctx := extensionRootContext()
		handled, err := root.BeforeParseRoute(ctx, args)
		require.True(t, handled)
		require.ErrorContains(t, err, "blocked by safety policy")
	}
}

func TestExtensionSafetyAllowedDispatchPreservesArgs(t *testing.T) {
	extensionPolicyHome(t, "*restore*", safety.ActionDeny)
	root, ctx := extensionRootContext()
	want := []string{"status", "--custom", "value", "--custom=other"}
	var got []string
	root.GetSubCommand("saectl").Run = func(_ *cli.Context, args []string) error {
		got = args
		return nil
	}
	root.Execute(ctx, append([]string{"saectl"}, want...))
	require.Equal(t, want, got)
}

func TestExtensionSafetyBlocksNestedExecution(t *testing.T) {
	extensionPolicyHome(t, "acrutil:diagnosis:restore", safety.ActionDeny)
	root, _ := extensionRootContext()
	called := false
	root.GetSubCommand("acrutil").GetSubCommand("diagnosis").Run = func(*cli.Context, []string) error {
		called = true
		return nil
	}
	cli.DisableExitCode()
	t.Cleanup(cli.EnableExitCode)
	var stderr bytes.Buffer
	ctx := cli.NewCommandContext(io.Discard, &stderr)
	ctx.EnterCommand(root)
	root.Execute(ctx, []string{"acrutil", "diagnosis", "restore"})
	require.False(t, called)
	require.Contains(t, stderr.String(), "blocked by safety policy")
}

func TestExtensionSafetyPreservesPreviousRouter(t *testing.T) {
	extensionPolicyHome(t, "*restore*", safety.ActionDeny)
	root, ctx := extensionRootContext()
	called := false
	root.BeforeParseRoute = func(_ *cli.Context, args []string) (bool, error) {
		called = true
		require.Equal(t, []string{"saectl", "status"}, args)
		return true, nil
	}
	cliext.AttachSafetyPolicy(root, map[string]bool{"saectl": true})
	handled, err := root.BeforeParseRoute(ctx, []string{"saectl", "restore"})
	require.True(t, handled)
	require.Error(t, err)
	require.False(t, called)
	handled, err = root.BeforeParseRoute(ctx, []string{"saectl", "status"})
	require.True(t, handled)
	require.NoError(t, err)
	require.True(t, called)
}

func TestExtensionSafetyPolicyFailuresAndOverrides(t *testing.T) {
	for _, failure := range []string{"invalid JSON", "unreadable policy", "invalid env"} {
		t.Run(failure, func(t *testing.T) {
			dir := extensionPolicyHome(t, "*", safety.ActionAllow)
			path := filepath.Join(dir, safety.SafetyPolicyFileName)
			switch failure {
			case "invalid JSON":
				require.NoError(t, os.WriteFile(path, []byte("{"), 0600))
			case "unreadable policy":
				require.NoError(t, os.Remove(path))
				require.NoError(t, os.Mkdir(path, 0700))
			case "invalid env":
				t.Setenv(safety.EnvSafetyPolicyEnabled, "bogus")
			}
			root, ctx := extensionRootContext()
			handled, err := root.BeforeParseRoute(ctx, []string{"saectl", "restore"})
			require.True(t, handled)
			require.ErrorContains(t, err, "load safety policy failed")
		})
	}
	t.Run("environment deny", func(t *testing.T) {
		extensionPolicyHome(t, "*", safety.ActionAllow)
		t.Setenv(safety.EnvSafetyPolicyRules, "*restore*=deny")
		root, ctx := extensionRootContext()
		_, err := root.BeforeParseRoute(ctx, []string{"saectl", "restore"})
		require.ErrorContains(t, err, "blocked by safety policy")
	})
	t.Run("disabled", func(t *testing.T) {
		extensionPolicyHome(t, "*", safety.ActionDeny)
		t.Setenv(safety.EnvSafetyPolicyEnabled, "false")
		root, ctx := extensionRootContext()
		handled, err := root.BeforeParseRoute(ctx, []string{"saectl", "restore"})
		require.False(t, handled)
		require.NoError(t, err)
	})
}

func TestExtensionSafetyConfirmation(t *testing.T) {
	for _, approval := range []string{"none", "--yes", "-y", "env"} {
		t.Run(approval, func(t *testing.T) {
			extensionPolicyHome(t, "*restore*", safety.ActionConfirm)
			oldStdin := os.Stdin
			f, err := os.CreateTemp(t.TempDir(), "stdin")
			require.NoError(t, err)
			os.Stdin = f
			t.Cleanup(func() { os.Stdin = oldStdin; f.Close() })
			args := []string{"saectl", "restore"}
			if approval == "env" {
				t.Setenv("ALIBABA_CLOUD_SAFETY_SKIP_CONFIRM", "true")
			} else if approval != "none" {
				args = append(args, approval)
			}
			root, ctx := extensionRootContext()
			handled, err := root.BeforeParseRoute(ctx, args)
			if approval == "none" {
				require.True(t, handled)
				require.ErrorContains(t, err, "requires confirmation")
			} else {
				require.False(t, handled)
				require.NoError(t, err)
			}
		})
	}
}
