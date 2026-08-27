package openapi

import (
	"io"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/aimode"
)

func TestCliAIOverridesForOpenAPIIncludesDetectedAgent(t *testing.T) {
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
