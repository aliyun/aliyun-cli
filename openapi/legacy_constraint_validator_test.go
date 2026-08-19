package openapi

import (
	"bytes"
	"errors"
	"path/filepath"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
)

func legacyConstraintContext(t *testing.T, forceOn, forceOff bool) *cli.Context {
	t.Helper()
	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	ctx.SetUnknownFlags(cli.NewFlagSet())
	ctx.Flags().Add(config.NewConfigurePathFlag())
	config.ConfigurePathFlag(ctx.Flags()).SetAssigned(true)
	config.ConfigurePathFlag(ctx.Flags()).SetValue(filepath.Join(t.TempDir(), "config.json"))
	ctx.Flags().Add(NewCliAIModeFlag())
	ctx.Flags().Add(NewCliNoAIModeFlag())
	if forceOn {
		CliAIModeFlag(ctx.Flags()).SetAssigned(true)
	}
	if forceOff {
		CliNoAIModeFlag(ctx.Flags()).SetAssigned(true)
	}
	return ctx
}

func assignLegacyUnknown(t *testing.T, ctx *cli.Context, name, value string) {
	t.Helper()
	flag, err := ctx.UnknownFlags().AddByName(name)
	if err != nil {
		t.Fatal(err)
	}
	flag.SetAssigned(true)
	flag.SetValue(value)
}

func legacyConstraintAPI() *canonicalmeta.API {
	return &canonicalmeta.API{Parameters: []canonicalmeta.Parameter{
		{
			Name: "mode", RawName: "Mode", Type: "string", Location: "query",
			Enum: []string{"ReadOnly", "ReadWrite"},
		},
		{
			Name: "count", RawName: "Count", Type: "integer", Location: "query",
			Minimum: "1", Maximum: "3",
		},
		{
			Name: "tag", RawName: "Tag", Type: "array", Location: "query",
			Element: &canonicalmeta.TypeShape{Type: "object", Fields: []canonicalmeta.Field{{
				Name: "key", RawName: "Key", Type: "string",
				Pattern: "^[a-z]+$",
			}}},
		},
		{
			Name: "zone", RawName: "Zone", Type: "array", Location: "query",
			Element: &canonicalmeta.TypeShape{Type: "string", Enum: []string{"a", "b"}},
		},
	}}
}

func TestValidateLegacyConstraintsPascalCaseAndNestedLeaf(t *testing.T) {
	tests := []struct {
		name       string
		flag       string
		value      string
		constraint string
	}{
		{name: "scalar enum is case sensitive", flag: "Mode", value: "readonly", constraint: "enum"},
		{name: "integer lower bound is inclusive", flag: "Count", value: "0", constraint: "minimum"},
		{name: "nested repeat-list leaf", flag: "Tag.1.Key", value: "NOT_LOWER", constraint: "pattern"},
		{name: "scalar repeat-list element", flag: "Zone.1", value: "c", constraint: "enum"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := legacyConstraintContext(t, true, false)
			assignLegacyUnknown(t, ctx, tt.flag, tt.value)
			err := validateLegacyConstraints(ctx, legacyConstraintAPI())
			var violation *ConstraintViolationError
			if !errors.As(err, &violation) {
				t.Fatalf("error = %v, want ConstraintViolationError", err)
			}
			if violation.Flag != tt.flag || violation.Constraint != tt.constraint {
				t.Fatalf("violation = %#v", violation)
			}
		})
	}

	for _, value := range []string{"1", "3"} {
		ctx := legacyConstraintContext(t, true, false)
		assignLegacyUnknown(t, ctx, "Count", value)
		if err := validateLegacyConstraints(ctx, legacyConstraintAPI()); err != nil {
			t.Fatalf("inclusive bound rejected %q: %v", value, err)
		}
	}
}

func TestValidateLegacyConstraintsNumericBoundsAreExact(t *testing.T) {
	param := canonicalmeta.Parameter{
		Name: "sequence", RawName: "Sequence", Type: "integer", Location: "query",
		Minimum: "9223372036854775809", Maximum: "9223372036854775811",
	}
	api := &canonicalmeta.API{Parameters: []canonicalmeta.Parameter{param}}
	for _, value := range []string{"9223372036854775809", "9223372036854775811"} {
		ctx := legacyConstraintContext(t, true, false)
		assignLegacyUnknown(t, ctx, "Sequence", value)
		if err := validateLegacyConstraints(ctx, api); err != nil {
			t.Fatalf("inclusive exact bound rejected %q: %v", value, err)
		}
	}

	ctx := legacyConstraintContext(t, true, false)
	assignLegacyUnknown(t, ctx, "Sequence", "9223372036854775808")
	err := validateLegacyConstraints(ctx, api)
	var violation *ConstraintViolationError
	if !errors.As(err, &violation) || violation.Constraint != "minimum" {
		t.Fatalf("error = %#v, want minimum violation", err)
	}
}

func TestValidateLegacyConstraintsAIModeOffAndForceOff(t *testing.T) {
	for _, tc := range []struct {
		name     string
		forceOn  bool
		forceOff bool
	}{
		{name: "AI mode off"},
		{name: "force off wins", forceOn: true, forceOff: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := legacyConstraintContext(t, tc.forceOn, tc.forceOff)
			assignLegacyUnknown(t, ctx, "Mode", "invalid")
			if err := validateLegacyConstraints(ctx, legacyConstraintAPI()); err != nil {
				t.Fatalf("validation should be disabled: %v", err)
			}
		})
	}
}

func TestValidateLegacyConstraintsFailOpenInvalidSchemaAndUnknownFlag(t *testing.T) {
	api := &canonicalmeta.API{Parameters: []canonicalmeta.Parameter{
		{
			Name: "value", RawName: "Value", Type: "integer", Location: "query",
			Minimum: "not-a-number", Maximum: "also-invalid",
		},
		{
			Name: "text", RawName: "Text", Type: "string", Location: "query",
			Pattern: "[",
		},
		{
			Name: "body", RawName: "body", Type: "string", Location: "body",
			Enum: []string{"schema-body"},
		},
	}}
	ctx := legacyConstraintContext(t, true, false)
	assignLegacyUnknown(t, ctx, "Value", "2")
	assignLegacyUnknown(t, ctx, "Text", "anything")
	assignLegacyUnknown(t, ctx, "body", `{"raw":true}`)
	assignLegacyUnknown(t, ctx, "Unknown", "anything")
	if err := validateLegacyConstraints(ctx, api); err != nil {
		t.Fatalf("invalid producer schema, raw body and unknown flags must fail open: %v", err)
	}
}
