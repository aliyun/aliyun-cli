package openapi

import (
	"bytes"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
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
			Name: "name", RawName: "Name", Type: "string", Location: "query",
			MinLength: "2", MaxLength: "3",
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
		{name: "string minimum length", flag: "Name", value: "云", constraint: "minLength"},
		{name: "string maximum length", flag: "Name", value: "阿里云CLI", constraint: "maxLength"},
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
	for _, value := range []string{"阿里", "a云c"} {
		ctx := legacyConstraintContext(t, true, false)
		assignLegacyUnknown(t, ctx, "Name", value)
		if err := validateLegacyConstraints(ctx, legacyConstraintAPI()); err != nil {
			t.Fatalf("inclusive Unicode length rejected %q: %v", value, err)
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
			MinLength: "bad", MaxLength: "-1", Pattern: "[",
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

func TestConstraintViolationErrorMessages(t *testing.T) {
	tests := []struct {
		violation  ConstraintViolationError
		wantPhrase string
	}{
		{ConstraintViolationError{Flag: "Mode", Value: "bad", Constraint: "enum"}, "not allowed"},
		{ConstraintViolationError{Flag: "Count", Value: "0", Constraint: "minimum", Minimum: "1"}, "greater than or equal to 1"},
		{ConstraintViolationError{Flag: "Count", Value: "4", Constraint: "maximum", Maximum: "3"}, "less than or equal to 3"},
		{ConstraintViolationError{Flag: "Name", Value: "a", Constraint: "minLength", MinLength: "2"}, "at least 2 characters"},
		{ConstraintViolationError{Flag: "Name", Value: "abcd", Constraint: "maxLength", MaxLength: "3"}, "at most 3 characters"},
		{ConstraintViolationError{Flag: "Name", Value: "A", Constraint: "pattern", Pattern: "^[a-z]+$"}, "does not match pattern"},
		{ConstraintViolationError{Flag: "Value", Value: "x", Constraint: "future"}, "violates its schema constraint"},
	}
	for _, test := range tests {
		if got := test.violation.Error(); !strings.Contains(got, test.wantPhrase) {
			t.Errorf("Error() = %q, want phrase %q", got, test.wantPhrase)
		}
	}

	docRequired := &LegacyDocRequiredError{Flags: []string{"--Name", "--RegionId"}}
	if got := docRequired.Error(); got != "missing required parameter(s): --Name, --RegionId" {
		t.Fatalf("LegacyDocRequiredError.Error() = %q", got)
	}
	docRequired.AIRecoveryEligible()
}

func TestValidateLegacyDocRequiredScalarAndRepeatLists(t *testing.T) {
	api := &canonicalmeta.API{Parameters: []canonicalmeta.Parameter{
		{Name: "name", RawName: "Name", Type: "string", Location: "query", DocRequired: true},
		{Name: "tags", RawName: "Tag", Type: "array", Location: "query", ParamStyle: "repeatList", DocRequired: true,
			Element: &canonicalmeta.TypeShape{Type: "object", Fields: []canonicalmeta.Field{
				{Name: "key", RawName: "Key", Type: "string", DocRequired: true},
				{Name: "value", RawName: "Value", Type: "string"},
			}}},
		{Name: "zones", RawName: "Zone", Type: "array", Location: "query", ParamStyle: "repeatList", DocRequired: true,
			Element: &canonicalmeta.TypeShape{Type: "string"}},
	}}

	t.Run("all documentation-required paths are reported", func(t *testing.T) {
		ctx := legacyConstraintContext(t, true, false)
		err := validateLegacyDocRequired(ctx, api)
		var missing *LegacyDocRequiredError
		if !errors.As(err, &missing) {
			t.Fatalf("error = %v, want LegacyDocRequiredError", err)
		}
		assertStringSliceEqual(t, []string{"--Name", "--Tag", "--Zone"}, missing.Flags)
	})

	t.Run("child paths are checked for every assigned instance", func(t *testing.T) {
		ctx := legacyConstraintContext(t, true, false)
		assignLegacyUnknown(t, ctx, "Name", "demo")
		assignLegacyUnknown(t, ctx, "Tag.2.Value", "value")
		assignLegacyUnknown(t, ctx, "Tag.10.Key", "key")
		assignLegacyUnknown(t, ctx, "Zone.1", "cn-a")
		err := validateLegacyDocRequired(ctx, api)
		var missing *LegacyDocRequiredError
		if !errors.As(err, &missing) {
			t.Fatalf("error = %v, want LegacyDocRequiredError", err)
		}
		assertStringSliceEqual(t, []string{"--Tag.2.Key"}, missing.Flags)
	})

	t.Run("all assigned values pass", func(t *testing.T) {
		ctx := legacyConstraintContext(t, true, false)
		assignLegacyUnknown(t, ctx, "Name", "demo")
		assignLegacyUnknown(t, ctx, "Tag.1.Key", "key")
		assignLegacyUnknown(t, ctx, "Zone.1", "cn-a")
		if err := validateLegacyDocRequired(ctx, api); err != nil {
			t.Fatalf("validateLegacyDocRequired() = %v", err)
		}
	})

	t.Run("assigned empty scalar stays missing", func(t *testing.T) {
		ctx := legacyConstraintContext(t, true, false)
		assignLegacyUnknown(t, ctx, "Name", "")
		err := validateLegacyDocRequired(ctx, &canonicalmeta.API{Parameters: api.Parameters[:1]})
		if err == nil || !strings.Contains(err.Error(), "--Name") {
			t.Fatalf("error = %v", err)
		}
	})
}

func TestValidateLegacyDocRequiredEarlyReturnsAndRawBody(t *testing.T) {
	ctx := legacyConstraintContext(t, true, false)
	api := &canonicalmeta.API{Parameters: []canonicalmeta.Parameter{{RawName: "Payload", Type: "string", Location: "body", DocRequired: true}}}
	if err := validateLegacyDocRequired(nil, api); err != nil {
		t.Fatalf("nil context: %v", err)
	}
	if err := validateLegacyDocRequired(ctx, nil); err != nil {
		t.Fatalf("nil API: %v", err)
	}
	ctxWithoutUnknown := cli.NewCommandContext(&bytes.Buffer{}, &bytes.Buffer{})
	if err := validateLegacyDocRequired(ctxWithoutUnknown, api); err != nil {
		t.Fatalf("nil unknown flags: %v", err)
	}
	off := legacyConstraintContext(t, false, false)
	if err := validateLegacyDocRequired(off, api); err != nil {
		t.Fatalf("AI mode off: %v", err)
	}

	ctx.Flags().Add(NewBodyFlag())
	BodyFlag(ctx.Flags()).SetAssigned(true)
	BodyFlag(ctx.Flags()).SetValue(`{"payload":true}`)
	if err := validateLegacyDocRequired(ctx, api); err != nil {
		t.Fatalf("raw body should satisfy body documentation requirements: %v", err)
	}
}

func TestLegacyRepeatListAndInvokerHelpers(t *testing.T) {
	assigned := map[string]bool{
		"Tag.10.Key": true, "Tag.2.Value": true, "Tag.2.Key": true,
		"Tag.bad.Key": true, "Tag.": true, "Other.1": true,
	}
	assertStringSliceEqual(t, []string{"Tag.10", "Tag.2"}, legacyRepeatListInstances("Tag", assigned))
	for _, value := range []string{"0", "1", "123"} {
		if !isDecimalIndex(value) {
			t.Errorf("isDecimalIndex(%q) = false", value)
		}
	}
	for _, value := range []string{"", "-1", "1a", "a"} {
		if isDecimalIndex(value) {
			t.Errorf("isDecimalIndex(%q) = true", value)
		}
	}

	api := &canonicalmeta.API{Name: "Demo"}
	if got := legacyAPIForInvoker(&RpcInvoker{api: api}); got != api {
		t.Fatalf("RPC API = %#v", got)
	}
	if got := legacyAPIForInvoker(&RestfulInvoker{api: api}); got != api {
		t.Fatalf("REST API = %#v", got)
	}
	if got := legacyAPIForInvoker(nil); got != nil {
		t.Fatalf("unknown invoker API = %#v", got)
	}

	for _, name := range []string{"body", "BODY", "body-file", "BODY-FILE"} {
		if !isRawBodyFlag(name) {
			t.Errorf("isRawBodyFlag(%q) = false", name)
		}
	}
	if isRawBodyFlag("payload") {
		t.Fatal("payload must not be treated as raw body")
	}
}

func assertStringSliceEqual(t *testing.T, want, got []string) {
	t.Helper()
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
