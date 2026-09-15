package openapi

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/aimode"
	"github.com/aliyun/aliyun-openapi-runtime/argparser"
	"github.com/aliyun/aliyun-openapi-runtime/engine"
	"github.com/stretchr/testify/assert"
)

func TestCliAIOverridesForOpenAPIIncludesDetectedAgent(t *testing.T) {
	t.Setenv(aimode.EnvAIMode, "")
	t.Setenv(aimode.EnvAgentIntegration, "")
	ctx := cli.NewCommandContext(io.Discard, io.Discard)
	AddFlags(ctx.Flags())

	on, off := CliAIOverridesForOpenAPI(ctx)
	if on || off {
		t.Fatalf("ordinary context overrides = %v, %v", on, off)
	}

	ctx.SetAgentName("codex")
	on, off = CliAIOverridesForOpenAPI(ctx)
	if !on || off {
		t.Fatalf("agent context overrides = %v, %v", on, off)
	}

	t.Setenv(aimode.EnvAIMode, "0")
	on, off = CliAIOverridesForOpenAPI(ctx)
	if on || off {
		t.Fatalf("environment opt-out did not suppress agent override = %v, %v", on, off)
	}

	t.Setenv(aimode.EnvAIMode, "")
	t.Setenv(aimode.EnvAgentIntegration, "disabled")
	on, off = CliAIOverridesForOpenAPI(ctx)
	if on || off {
		t.Fatalf("disabled Agent integration did not suppress agent override = %v, %v", on, off)
	}

	CliAIModeFlag(ctx.Flags()).SetAssigned(true)
	on, off = CliAIOverridesForOpenAPI(ctx)
	if !on || off {
		t.Fatalf("explicit force-on did not override environment opt-out = %v, %v", on, off)
	}

	CliNoAIModeFlag(ctx.Flags()).SetAssigned(true)
	on, off = CliAIOverridesForOpenAPI(ctx)
	if on || !off {
		t.Fatalf("agent force-off overrides = %v, %v", on, off)
	}

	on, off = CliAIOverridesForOpenAPI(nil)
	if on || off {
		t.Fatalf("nil context overrides = %v, %v", on, off)
	}
}

func TestDetectedAgentEnablesLegacyOpenAPIAIMode(t *testing.T) {
	ctx := legacyConstraintContext(t, false, false)
	ctx.SetAgentName("codex")
	assignLegacyUnknown(t, ctx, "Mode", "invalid")
	if err := validateLegacyConstraints(ctx, legacyConstraintAPI()); err == nil {
		t.Fatal("legacy constraints were not enabled for detected agent")
	}

	forceOff := legacyConstraintContext(t, false, true)
	forceOff.SetAgentName("codex")
	assignLegacyUnknown(t, forceOff, "Mode", "invalid")
	if err := validateLegacyConstraints(forceOff, legacyConstraintAPI()); err != nil {
		t.Fatalf("force-off did not disable agent legacy constraints: %v", err)
	}
}

func TestDetectedAgentAddsLegacyOpenAPIUserAgent(t *testing.T) {
	ctx := legacyConstraintContext(t, false, false)
	ctx.SetAgentName("codex")
	if suffix := aiModeSuffixForContext(ctx); suffix != aimode.UserAgentEnabledMarker {
		t.Fatalf("agent legacy OpenAPI suffix = %q", suffix)
	}

	CliNoAIModeFlag(ctx.Flags()).SetAssigned(true)
	if suffix := aiModeSuffixForContext(ctx); suffix != "" {
		t.Fatalf("force-off legacy OpenAPI suffix = %q", suffix)
	}
}

// TestAIModeEffectivePrecedenceChain locks the documented AI-mode precedence:
// --no-cli-ai-mode > --cli-ai-mode > ALIBABA_CLOUD_CLI_AI_MODE > agent
// auto-detection (gated by ALIBABA_CLOUD_CLI_AGENT_INTEGRATION) > ai-mode.json.
// The contract is frozen by test so later refactors cannot silently reorder it.
func TestAIModeEffectivePrecedenceChain(t *testing.T) {
	newCtx := func(agent bool) *cli.Context {
		ctx := cli.NewCommandContext(io.Discard, io.Discard)
		AddFlags(ctx.Flags())
		if agent {
			ctx.SetAgentName("codex")
		}
		return ctx
	}
	enabledFor := func(ctx *cli.Context, cfg *aimode.AiConfig) bool {
		on, off := CliAIOverridesForOpenAPI(ctx)
		return aimode.EnabledForCommand(cfg, on, off)
	}

	t.Run("default off", func(t *testing.T) {
		t.Setenv(aimode.EnvAIMode, "")
		t.Setenv(aimode.EnvAgentIntegration, "")
		assert.False(t, enabledFor(newCtx(false), &aimode.AiConfig{Enabled: false}))
	})

	t.Run("config file on", func(t *testing.T) {
		t.Setenv(aimode.EnvAIMode, "")
		t.Setenv(aimode.EnvAgentIntegration, "")
		assert.True(t, enabledFor(newCtx(false), &aimode.AiConfig{Enabled: true}))
	})

	t.Run("env on beats config off", func(t *testing.T) {
		t.Setenv(aimode.EnvAIMode, "1")
		assert.True(t, enabledFor(newCtx(false), &aimode.AiConfig{Enabled: false}))
	})

	t.Run("env off beats config on", func(t *testing.T) {
		t.Setenv(aimode.EnvAIMode, "0")
		assert.False(t, enabledFor(newCtx(false), &aimode.AiConfig{Enabled: true}))
	})

	t.Run("agent auto-detection turns on", func(t *testing.T) {
		t.Setenv(aimode.EnvAIMode, "")
		t.Setenv(aimode.EnvAgentIntegration, "")
		assert.True(t, enabledFor(newCtx(true), &aimode.AiConfig{Enabled: false}))
	})

	t.Run("agent auto-detection gated off by integration switch", func(t *testing.T) {
		t.Setenv(aimode.EnvAIMode, "")
		t.Setenv(aimode.EnvAgentIntegration, "disabled")
		assert.False(t, enabledFor(newCtx(true), &aimode.AiConfig{Enabled: false}))
	})

	t.Run("env opt-out beats agent detection", func(t *testing.T) {
		t.Setenv(aimode.EnvAIMode, "0")
		assert.False(t, enabledFor(newCtx(true), &aimode.AiConfig{Enabled: false}))
	})

	t.Run("flag on beats env off", func(t *testing.T) {
		t.Setenv(aimode.EnvAIMode, "0")
		ctx := newCtx(false)
		CliAIModeFlag(ctx.Flags()).SetAssigned(true)
		assert.True(t, enabledFor(ctx, &aimode.AiConfig{Enabled: false}))
	})

	t.Run("flag off beats everything", func(t *testing.T) {
		t.Setenv(aimode.EnvAIMode, "1")
		t.Setenv(aimode.EnvAgentIntegration, "")
		ctx := newCtx(true)
		CliNoAIModeFlag(ctx.Flags()).SetAssigned(true)
		assert.False(t, enabledFor(ctx, &aimode.AiConfig{Enabled: true}))
	})

	t.Run("flag on beats integration gate", func(t *testing.T) {
		t.Setenv(aimode.EnvAIMode, "")
		t.Setenv(aimode.EnvAgentIntegration, "disabled")
		ctx := newCtx(true)
		CliAIModeFlag(ctx.Flags()).SetAssigned(true)
		assert.True(t, enabledFor(ctx, &aimode.AiConfig{Enabled: false}))
	})
}

// TestExplicitOptOutKeepsHumanErrorUnderAgentDetection covers the opt-out path
// end to end: a detected agent environment plus --no-cli-ai-mode renders the
// human error, not the JSON envelope.
func TestExplicitOptOutKeepsHumanErrorUnderAgentDetection(t *testing.T) {
	t.Setenv(aimode.EnvAIMode, "")
	t.Setenv(aimode.EnvAgentIntegration, "")
	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	cmd := &cli.Command{Name: "aliyun", EnableUnknownFlag: true}
	config.AddFlags(cmd.Flags())
	AddFlags(cmd.Flags())
	ctx.EnterCommand(cmd)
	ctx.SetAgentName("codex")

	commando := &Commando{profile: config.Profile{Language: "en"}}
	cause := &engine.UsageError{
		Code: "UNKNOWN_FLAG",
		Err: fmt.Errorf("%w (run `aliyun ecs describe-instances --help` for accepted flags)",
			&argparser.UnknownFlagError{Flag: "Profile", Known: []string{"instance-type"}}),
	}
	got := commando.finishCommandRun(ctx, []string{"ecs", "describe-instances", "--no-cli-ai-mode", "--Profile", "default"}, cause)
	var agentErr *cli.AgentError
	assert.False(t, errors.As(got, &agentErr), "opt-out must not produce a JSON envelope")
	assert.Contains(t, got.Error(), "did you mean --profile")
}
