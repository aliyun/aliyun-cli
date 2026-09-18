package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/sysconfig/aimode"
)

func TestStartupAIModePrecedence(t *testing.T) {
	for _, tc := range []struct {
		name, env, integration string
		agent, saved           bool
		args                   []string
		want                   bool
	}{
		{name: "ordinary"},
		{name: "positional is not a flag", args: []string{"oss", "cat", "cli-ai-mode"}},
		{name: "saved", saved: true, want: true},
		{name: "environment", env: "1", want: true},
		{name: "environment off overrides saved", env: "0", saved: true},
		{name: "explicit on overrides env", env: "0", args: []string{"ecs", "--cli-ai-mode"}, want: true},
		{name: "explicit off wins", env: "1", args: []string{"--cli-ai-mode", "--no-cli-ai-mode"}},
		{name: "agent", agent: true, want: true},
		{name: "agent integration disabled", agent: true, integration: "disabled"},
		{name: "disabled integration preserves saved", agent: true, integration: "disabled", saved: true, want: true},
		{name: "env off overrides agent", agent: true, env: "0"},
		{name: "flag off overrides agent", agent: true, args: []string{"--no-cli-ai-mode"}},
		{name: "terminator", args: []string{"oss", "cat", "--", "--cli-ai-mode"}},
		{name: "flag value", args: []string{"--profile", "--cli-ai-mode"}},
		{name: "inline value", args: []string{"--profile=--cli-ai-mode"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			clearAgentDetectionEnv(t)
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("HOMEDRIVE", "")
			t.Setenv("HOMEPATH", "")
			t.Setenv("USERPROFILE", home)
			t.Setenv(aimode.EnvAIMode, tc.env)
			t.Setenv(aimode.EnvAgentIntegration, tc.integration)
			if tc.agent {
				t.Setenv("CODEX_SHELL", "1")
			}
			if tc.saved {
				dir := filepath.Join(home, ".aliyun")
				if err := os.MkdirAll(dir, 0700); err != nil {
					t.Fatal(err)
				}
				if err := aimode.Save(dir, &aimode.AiConfig{Enabled: true}); err != nil {
					t.Fatal(err)
				}
			}
			if got := startupAIMode(tc.args); got != tc.want {
				t.Fatalf("enabled=%v, want %v", got, tc.want)
			}
		})
	}
}

func TestStartupAIModeCustomConfigPath(t *testing.T) {
	clearAgentDetectionEnv(t)
	dir := t.TempDir()
	if err := aimode.Save(dir, &aimode.AiConfig{Enabled: true}); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"--config-path", filepath.Join(dir, "config.json")},
		{"--config-path=" + filepath.Join(dir, "config.json")},
	} {
		if !startupAIMode(args) {
			t.Fatalf("saved AI mode not loaded for %v", args)
		}
	}
}
