package cliext

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/safety"
)

func safetyContext(t *testing.T) *cli.Context {
	t.Helper()
	configPath := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("ALIBABA_CLOUD_SAFETY_RULES", "")
	t.Setenv("ALIBABA_CLOUD_SAFETY_ENABLED", "")
	t.Setenv("ALIBABA_CLOUD_SAFETY_SKIP_CONFIRM", "")
	ctx := cli.NewCommandContext(io.Discard, io.Discard)
	config.AddFlags(ctx.Flags())
	pathFlag := ctx.Flags().Get(config.ConfigurePathFlagName)
	pathFlag.SetValue(configPath)
	pathFlag.SetAssigned(true)
	ctx.Flags().Add(&cli.Flag{Name: "yes", Shorthand: 'y', AssignedMode: cli.AssignedNone})
	ctx.Flags().Add(&cli.Flag{Name: "version", AssignedMode: cli.AssignedOnce})
	return ctx
}

func TestExtensionSafetyPolicyRouting(t *testing.T) {
	for _, tc := range []struct {
		name   string
		args   []string
		denied bool
	}{
		{"nested", []string{"saectl", "delete", "app"}, true},
		{"root config", []string{"--region", "cn-hangzhou", "saectl", "delete", "app"}, true},
		{"host config removed", []string{"saectl", "delete", "-p", "profile", "app", "--region=cn-hangzhou"}, true},
		{"provider version flag", []string{"saectl", "delete", "--version", "app"}, true},
		{"literal command", []string{"saectl", "--", "delete", "app"}, true},
		{"unrelated", []string{"ecs", "delete", "app"}, false},
		{"empty", nil, false},
		{"syntax delegated", []string{"--region"}, false},
		{"no operation", []string{"saectl", "--profile", "default"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := safetyContext(t)
			if err := safety.SavePolicy(config.GetConfigDir(ctx), &safety.Policy{Enabled: true, Rules: []safety.Rule{{Pattern: "saectl:delete:app", Action: safety.ActionDeny}}}); err != nil {
				t.Fatal(err)
			}
			before := append([]string(nil), ctx.Flags().Get("region").GetValues()...)
			err := checkExtensionSafetyPolicy(ctx, tc.args, map[string]bool{"saectl": true})
			if (err != nil) != tc.denied {
				t.Fatalf("policy error = %v, denied = %v", err, tc.denied)
			}
			if !reflect.DeepEqual(before, ctx.Flags().Get("region").GetValues()) {
				t.Fatal("preflight changed original parser state")
			}
		})
	}
}

func TestExtensionSafetyIntrospectionAndBypass(t *testing.T) {
	for _, args := range [][]string{{"saectl"}, {"saectl", "help"}, {"saectl", "--help"}, {"saectl", "-h"}, {"saectl", "version"}, {"saectl", "--version"}, {"saectl", "--custom", "--help"}, {"saectl", "delete", "--help"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			ctx := safetyContext(t)
			if err := safety.SavePolicy(config.GetConfigDir(ctx), &safety.Policy{Enabled: true, Rules: []safety.Rule{{Pattern: "saectl:*", Action: safety.ActionDeny}}}); err != nil {
				t.Fatal(err)
			}
			err := checkExtensionSafetyPolicy(ctx, args, map[string]bool{"saectl": true})
			wantDeny := len(args) > 2
			if (err != nil) != wantDeny {
				t.Fatalf("error = %v, denied = %v", err, wantDeny)
			}
		})
	}
}

func TestExtensionSafetyApprovalAndConfigPath(t *testing.T) {
	for _, tc := range []struct {
		name  string
		extra []string
		env   string
	}{
		{"long", []string{"--yes"}, ""}, {"short", []string{"-y"}, ""}, {"environment", nil, "TrUe"}, {"environment numeric", nil, "1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := safetyContext(t)
			dir := t.TempDir()
			if err := safety.SavePolicy(dir, &safety.Policy{Enabled: true, Rules: []safety.Rule{{Pattern: "saectl:*", Action: safety.ActionConfirm}}}); err != nil {
				t.Fatal(err)
			}
			t.Setenv("ALIBABA_CLOUD_SAFETY_SKIP_CONFIRM", tc.env)
			args := append([]string{"saectl", "delete", "--config-path", filepath.Join(dir, "config.json")}, tc.extra...)
			if err := checkExtensionSafetyPolicy(ctx, args, map[string]bool{"saectl": true}); err != nil {
				t.Fatal(err)
			}
		})
	}
	ctx := safetyContext(t)
	if err := checkExtensionSafetyPolicy(ctx, []string{"saectl", "delete", "--config-path"}, map[string]bool{"saectl": true}); err == nil || !strings.Contains(err.Error(), "missing value") {
		t.Fatalf("missing config value = %v", err)
	}
	dir := config.GetConfigDir(ctx)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(safety.GetPolicyFilePath(dir), []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := checkExtensionSafetyPolicy(ctx, []string{"saectl", "delete"}, map[string]bool{"saectl": true}); err == nil || !strings.Contains(err.Error(), "load safety policy failed") {
		t.Fatalf("invalid policy = %v", err)
	}
	ctx.SetCompletion(&cli.Completion{})
	if err := checkExtensionSafetyPolicy(ctx, []string{"saectl", "delete"}, map[string]bool{"saectl": true}); err != nil {
		t.Fatalf("completion checked policy: %v", err)
	}
}

func TestAttachSafetyPolicyChainsRoute(t *testing.T) {
	ctx := safetyContext(t)
	sentinel := errors.New("route failed")
	called := 0
	root := &cli.Command{BeforeParseRoute: func(*cli.Context, []string) (bool, error) { called++; return true, sentinel }}
	AttachSafetyPolicy(root, map[string]bool{"saectl": true})
	handled, err := root.BeforeParseRoute(ctx, []string{"ecs", "DescribeInstances"})
	if !handled || !errors.Is(err, sentinel) || called != 1 {
		t.Fatalf("previous route = %v, %v, %d", handled, err, called)
	}
	if err := safety.SavePolicy(config.GetConfigDir(ctx), &safety.Policy{Enabled: true, Rules: []safety.Rule{{Pattern: "saectl:*", Action: safety.ActionDeny}}}); err != nil {
		t.Fatal(err)
	}
	handled, err = root.BeforeParseRoute(ctx, []string{"saectl", "delete"})
	if !handled || err == nil || called != 1 {
		t.Fatalf("denied route = %v, %v, %d", handled, err, called)
	}
	root = &cli.Command{}
	AttachSafetyPolicy(root, nil)
	handled, err = root.BeforeParseRoute(ctx, nil)
	if handled || err != nil {
		t.Fatalf("unhandled route = %v, %v", handled, err)
	}
}
