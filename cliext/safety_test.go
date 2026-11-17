package cliext

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/safety"
	"github.com/stretchr/testify/require"
)

func safetyContext(t *testing.T, action safety.Action) (*cli.Command, *cli.Context, string) {
	t.Helper()
	for _, name := range []string{safety.EnvSafetyPolicyEnabled, safety.EnvSafetyPolicyRules, "ALIBABA_CLOUD_SAFETY_SKIP_CONFIRM"} {
		t.Setenv(name, "")
		require.NoError(t, os.Unsetenv(name))
	}
	dir := t.TempDir()
	require.NoError(t, safety.SavePolicy(dir, &safety.Policy{Enabled: true, Rules: []safety.Rule{{Pattern: "provider:delete*", Action: action}}}))
	root := &cli.Command{Name: "aliyun"}
	config.AddFlags(root.Flags())
	root.Flags().Add(&cli.Flag{Name: "yes", Shorthand: 'y', AssignedMode: cli.AssignedNone})
	root.Flags().Add(&cli.Flag{Name: "version", AssignedMode: cli.AssignedOnce})
	ctx := cli.NewCommandContext(io.Discard, io.Discard)
	ctx.EnterCommand(root)
	config.ConfigurePathFlag(ctx.Flags()).SetValue(filepath.Join(dir, "config.json"))
	config.ConfigurePathFlag(ctx.Flags()).SetAssigned(true)
	return root, ctx, dir
}

func TestSafetyPreflightPreservesProviderArgumentsAndHostFlags(t *testing.T) {
	for _, args := range [][]string{
		{"provider", "delete"},
		{"--profile", "demo", "provider", "delete"},
		{"provider", "--profile=demo", "delete"},
		{"provider", "--profile:demo", "delete"},
		{"provider", "--force", "delete"},
		{"provider", "--version", "delete"},
		{"provider", "delete", "--name", "--help"},
		{"provider", "--", "delete", "--help"},
		{"provider", "delete", "--yes"},
		{"provider", "delete", "-y"},
	} {
		t.Run(args[len(args)-1], func(t *testing.T) {
			root, ctx, _ := safetyContext(t, safety.ActionDeny)
			AttachSafetyPolicy(root, map[string]bool{"provider": true})
			original := append([]string(nil), args...)
			handled, err := root.BeforeParseRoute(ctx, args)
			require.True(t, handled)
			require.ErrorContains(t, err, "provider")
			require.ErrorContains(t, err, "delete")
			require.Equal(t, original, args)
			require.False(t, ctx.Flags().Get("yes").IsAssigned())
			require.False(t, config.ProfileFlag(ctx.Flags()).IsAssigned())
		})
	}
}

func TestSafetyPreflightAllowsIntrospectionAndUnrelatedRoutes(t *testing.T) {
	for _, args := range [][]string{
		{}, {"--profile"}, {"other", "delete"}, {"provider"}, {"provider", "status"},
		{"provider", "help"}, {"provider", "--help"}, {"provider", "-h"}, {"provider", "version"}, {"provider", "--version"},
	} {
		root, ctx, _ := safetyContext(t, safety.ActionDeny)
		AttachSafetyPolicy(root, map[string]bool{"provider": true})
		handled, err := root.BeforeParseRoute(ctx, args)
		require.False(t, handled, "%v", args)
		require.NoError(t, err, "%v", args)
	}
	root, ctx, _ := safetyContext(t, safety.ActionDeny)
	ctx.SetCompletion(&cli.Completion{})
	AttachSafetyPolicy(root, map[string]bool{"provider": true})
	handled, err := root.BeforeParseRoute(ctx, []string{"provider", "delete"})
	require.False(t, handled)
	require.NoError(t, err)
}

func TestSafetyPreflightUsesExplicitConfigAndReportsInvalidPolicy(t *testing.T) {
	for _, inline := range []bool{false, true} {
		root, ctx, _ := safetyContext(t, safety.ActionAllow)
		dir := t.TempDir()
		require.NoError(t, safety.SavePolicy(dir, &safety.Policy{Enabled: true, Rules: []safety.Rule{{Pattern: "*", Action: safety.ActionDeny}}}))
		args := []string{"provider", "delete", "--config-path", filepath.Join(dir, "config.json")}
		if inline {
			args = []string{"provider", "delete", "--config-path=" + filepath.Join(dir, "config.json")}
		}
		AttachSafetyPolicy(root, map[string]bool{"provider": true})
		handled, err := root.BeforeParseRoute(ctx, args)
		require.True(t, handled)
		require.ErrorContains(t, err, "blocked")
	}
	root, ctx, dir := safetyContext(t, safety.ActionAllow)
	AttachSafetyPolicy(root, map[string]bool{"provider": true})
	_, err := root.BeforeParseRoute(ctx, []string{"provider", "delete", "--config-path"})
	require.ErrorContains(t, err, "missing value for --config-path")
	require.NoError(t, os.WriteFile(filepath.Join(dir, safety.SafetyPolicyFileName), []byte("{"), 0600))
	_, err = root.BeforeParseRoute(ctx, []string{"provider", "delete"})
	require.ErrorContains(t, err, "load safety policy failed")
}

func TestSafetyPreflightChainsPreviousRouterAfterApproval(t *testing.T) {
	for _, approval := range []string{"global", "long", "short", "env true", "env one"} {
		t.Run(approval, func(t *testing.T) {
			root, ctx, _ := safetyContext(t, safety.ActionConfirm)
			args := []string{"provider", "delete"}
			switch approval {
			case "global":
				args = append([]string{"--yes"}, args...)
			case "long":
				args = append(args, "--yes")
			case "short":
				args = append(args, "-y")
			case "env true":
				t.Setenv("ALIBABA_CLOUD_SAFETY_SKIP_CONFIRM", "TrUe")
			case "env one":
				t.Setenv("ALIBABA_CLOUD_SAFETY_SKIP_CONFIRM", "1")
			}
			called := false
			root.BeforeParseRoute = func(_ *cli.Context, got []string) (bool, error) {
				called = true
				require.Equal(t, args, got)
				return true, nil
			}
			AttachSafetyPolicy(root, map[string]bool{"provider": true})
			handled, err := root.BeforeParseRoute(ctx, args)
			require.True(t, handled)
			require.NoError(t, err)
			require.True(t, called)
		})
	}
	root, ctx, _ := safetyContext(t, safety.ActionDeny)
	root.BeforeParseRoute = func(*cli.Context, []string) (bool, error) {
		t.Fatal("denied command reached previous router")
		return false, nil
	}
	AttachSafetyPolicy(root, map[string]bool{"provider": true})
	_, err := root.BeforeParseRoute(ctx, []string{"provider", "delete"})
	require.Error(t, err)
}
