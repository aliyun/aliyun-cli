// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package openapi

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/meta"
	"github.com/aliyun/aliyun-openapi-runtime/argparser"
	"github.com/aliyun/aliyun-openapi-runtime/engine"
	"github.com/aliyun/aliyun-openapi-runtime/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func styleMigrationTestAPI() *canonicalmeta.API {
	return &canonicalmeta.API{
		Name:    "DescribeInstances",
		CmdName: "describe-instances",
		Parameters: []canonicalmeta.Parameter{
			{Name: "biz_region_id", RawName: "RegionId", Options: []string{"--biz-region-id"}, Location: "query"},
			{Name: "instance_id", RawName: "InstanceId", Options: []string{"--instance-id"}, Location: "query"},
			{Name: "x_trace", RawName: "XTrace", Options: []string{"--x-trace"}, Location: "header"},
		},
	}
}

func newAssignedFlag(name string, values ...string) *cli.Flag {
	f := &cli.Flag{Name: name, AssignedMode: cli.AssignedOnce}
	f.SetAssigned(true)
	for _, v := range values {
		f.SetValues(append(f.GetValues(), v))
	}
	if len(values) > 0 {
		f.SetValue(values[0])
	}
	return f
}

func TestStyleRenameMaps(t *testing.T) {
	kebabToRaw, rawToKebab := styleRenameMaps(styleMigrationTestAPI())
	assert.Equal(t, "RegionId", kebabToRaw["biz-region-id"])
	assert.Equal(t, "biz-region-id", rawToKebab["RegionId"])
	assert.Equal(t, "InstanceId", kebabToRaw["instance-id"])
	// header-position parameters are excluded from the suggestion space
	_, ok := kebabToRaw["x-trace"]
	assert.False(t, ok)

	k2r, r2k := styleRenameMaps(nil)
	assert.Nil(t, k2r)
	assert.Nil(t, r2k)
}

func TestInvalidParameterError_CrossStyleRenameSuggestion(t *testing.T) {
	err := NewInvalidParameterErrorFromCanonical("biz-region-id", styleMigrationTestAPI(), "ecs", cli.NewFlagSet())
	// The rename table maps the kebab flag to its PascalCase raw name; a plain
	// edit-distance matcher could never bridge "biz-region-id" and "RegionId".
	assert.Equal(t, []string{"RegionId"}, err.GetSuggestions())
	assert.Equal(t, []string{"--RegionId"}, err.AgentSuggestions())
}

func TestInvalidParameterError_TypoSuggestionStillWorks(t *testing.T) {
	err := NewInvalidParameterErrorFromCanonical("InstnaceId", styleMigrationTestAPI(), "ecs", cli.NewFlagSet())
	assert.Contains(t, err.GetSuggestions(), "InstanceId")
	assert.Contains(t, err.AgentSuggestions(), "--InstanceId")
}

func TestRebuildStyleEquivalentCommand(t *testing.T) {
	api := styleMigrationTestAPI()
	kebabToRaw, rawToKebab := styleRenameMaps(api)

	newContext := func() *cli.Context {
		ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
		ctx.Flags().Add(newAssignedFlag("region", "cn-hangzhou"))
		return ctx
	}

	t.Run("pascal to kebab maps raw names and keeps kebab flags", func(t *testing.T) {
		ctx := newContext()
		unknown := cli.NewFlagSet()
		unknown.Add(newAssignedFlag("biz-region-id", "cn-hangzhou"))
		unknown.Add(newAssignedFlag("InstanceId", "i-123"))
		ctx.SetUnknownFlags(unknown)
		got := rebuildStyleEquivalentCommand("ecs", "describe-instances", ctx, rawToKebab, keySet(kebabToRaw))
		assert.Equal(t, "aliyun ecs describe-instances --region cn-hangzhou --biz-region-id cn-hangzhou --instance-id i-123", got)
	})

	t.Run("kebab to pascal maps kebab names and keeps raw names", func(t *testing.T) {
		ctx := newContext()
		unknown := cli.NewFlagSet()
		unknown.Add(newAssignedFlag("biz-region-id", "cn-hangzhou"))
		ctx.SetUnknownFlags(unknown)
		got := rebuildStyleEquivalentCommand("ecs", "DescribeInstances", ctx, kebabToRaw, keySet(rawToKebab))
		assert.Equal(t, "aliyun ecs DescribeInstances --region cn-hangzhou --RegionId cn-hangzhou", got)
	})

	t.Run("valueless flags stay bare", func(t *testing.T) {
		ctx := newContext()
		unknown := cli.NewFlagSet()
		unknown.Add(newAssignedFlag("biz-region-id", "cn-hangzhou"))
		ctx.SetUnknownFlags(unknown)
		ctx.Flags().Add(newAssignedFlag("force"))
		got := rebuildStyleEquivalentCommand("ecs", "describe-instances", ctx, rawToKebab, keySet(kebabToRaw))
		assert.Equal(t, "aliyun ecs describe-instances --region cn-hangzhou --force --biz-region-id cn-hangzhou", got)
	})

	t.Run("values with spaces are quoted", func(t *testing.T) {
		ctx := newContext()
		unknown := cli.NewFlagSet()
		unknown.Add(newAssignedFlag("InstanceId", "hello world"))
		ctx.SetUnknownFlags(unknown)
		got := rebuildStyleEquivalentCommand("ecs", "describe-instances", ctx, rawToKebab, keySet(kebabToRaw))
		assert.Contains(t, got, "--instance-id 'hello world'")
	})

	t.Run("unmappable api flag fails closed", func(t *testing.T) {
		ctx := newContext()
		unknown := cli.NewFlagSet()
		unknown.Add(newAssignedFlag("totally-unknown", "x"))
		ctx.SetUnknownFlags(unknown)
		got := rebuildStyleEquivalentCommand("ecs", "describe-instances", ctx, rawToKebab, keySet(kebabToRaw))
		assert.Equal(t, "", got)
	})

	t.Run("FILE-suffixed flags fail closed", func(t *testing.T) {
		// --body-FILE reads the value from a file on the PascalCase chain;
		// the kebab engine has no per-parameter -FILE convention, so the
		// equivalent must not be rebuilt at all rather than lose the
		// read-from-file semantics.
		ctx := newContext()
		unknown := cli.NewFlagSet()
		unknown.Add(newAssignedFlag("biz-region-id", "cn-hangzhou"))
		unknown.Add(newAssignedFlag("body-FILE", "/tmp/x.json"))
		ctx.SetUnknownFlags(unknown)
		got := rebuildStyleEquivalentCommand("ecs", "describe-instances", ctx, rawToKebab, keySet(kebabToRaw))
		assert.Equal(t, "", got)
	})
}

// stubEngineServedCommands pins the engine-serving gate for deterministic
// tests; the real engine metadata is not available in unit test environments.
func stubEngineServedCommands(t *testing.T, served []string) {
	t.Helper()
	original := engineServedCommands
	engineServedCommands = func(string) []string { return served }
	t.Cleanup(func() { engineServedCommands = original })
}

func TestInvalidParameterError_StyleMigrationTipWithoutServedKebabCommand(t *testing.T) {
	// The engine serves nothing for this product: the tip degrades to naming
	// the style-correct flag only, and no unrunnable equivalent is advised.
	stubEngineServedCommands(t, nil)
	err := NewInvalidParameterErrorFromCanonical("biz-region-id", styleMigrationTestAPI(), "ecs", cli.NewFlagSet())
	err.attachStyleMigration(styleMigrationTestAPI(), cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer)))
	assert.Equal(t, "--biz-region-id is the kebab-style name of --RegionId; use --RegionId here.", err.styleMigrationTip())
	assert.Equal(t, "", err.equivalentCommand)

	plain := NewInvalidParameterErrorFromCanonical("InstnaceId", styleMigrationTestAPI(), "ecs", cli.NewFlagSet())
	plain.attachStyleMigration(styleMigrationTestAPI(), cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer)))
	assert.Equal(t, "", plain.styleMigrationTip())
}

func TestInvalidParameterError_StyleMigrationTipWithEquivalentCommand(t *testing.T) {
	// The engine serves the kebab command: the tip carries the rebuilt
	// equivalent command with every assigned flag translated.
	stubEngineServedCommands(t, []string{"describe-instances"})
	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	unknown := cli.NewFlagSet()
	unknown.Add(newAssignedFlag("biz-region-id", "cn-hangzhou"))
	unknown.Add(newAssignedFlag("InstanceId", "i-123"))
	ctx.SetUnknownFlags(unknown)

	err := NewInvalidParameterErrorFromCanonical("biz-region-id", styleMigrationTestAPI(), "ecs", cli.NewFlagSet())
	err.attachStyleMigration(styleMigrationTestAPI(), ctx)
	tip := err.styleMigrationTip()
	assert.Contains(t, tip, "Equivalent command:")
	assert.Contains(t, tip, "aliyun ecs describe-instances --biz-region-id cn-hangzhou --instance-id i-123")
}

func newStyleMixedTestCommando(t *testing.T) *Commando {
	t.Helper()
	repo := newFakeCanonicalRepo()
	repo.AddVersionIndex("ecs", "2014-05-26", &canonicalmeta.VersionIndex{APIs: map[string]canonicalmeta.VersionAPIEntry{
		"DescribeInstances": {CmdName: "describe-instances"},
	}})
	repo.AddAPI("ecs", "2014-05-26", styleMigrationTestAPI())
	products, err := meta.MockLoadRepository([]meta.Product{{Code: "ecs", Version: "2014-05-26"}})
	require.NoError(t, err)
	return &Commando{library: &Library{builtinRepo: products, canonicalRepo: repo}}
}

func unknownFlagUsageError(flag string, known ...string) error {
	return &engine.UsageError{
		Code: "UNKNOWN_FLAG",
		Err:  fmt.Errorf("%w (run `aliyun ecs describe-instances --help` for accepted flags)", &argparser.UnknownFlagError{Flag: flag, Known: known}),
	}
}

func TestAdaptStyleMixedFlagError(t *testing.T) {
	args := []string{"ecs", "describe-instances"}

	t.Run("pascal flag on kebab command gets suggestion and equivalent", func(t *testing.T) {
		c := newStyleMixedTestCommando(t)
		ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
		ctx.Flags().Add(newAssignedFlag("RegionId", "cn-hangzhou"))
		unknown := cli.NewFlagSet()
		unknown.Add(newAssignedFlag("instance-id", "i-123"))
		ctx.SetUnknownFlags(unknown)

		err := c.adaptStyleMixedFlagError(unknownFlagUsageError("RegionId", "biz-region-id", "instance-id"), args, ctx)
		var mixed *styleMixedFlagError
		require.ErrorAs(t, err, &mixed)
		assert.Equal(t, []string{"--biz-region-id"}, mixed.GetSuggestions())
		assert.Contains(t, mixed.Error(), "--RegionId is a PascalCase parameter name; kebab commands use --biz-region-id")
		assert.Equal(t, "aliyun ecs DescribeInstances --RegionId cn-hangzhou --InstanceId i-123", mixed.equivalent)
	})

	t.Run("unmappable flag passes through unchanged", func(t *testing.T) {
		c := newStyleMixedTestCommando(t)
		cause := unknownFlagUsageError("NotARealParam", "biz-region-id")
		err := c.adaptStyleMixedFlagError(cause, args, cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer)))
		assert.Equal(t, cause, err)
	})

	t.Run("rename not served by the engine fails closed", func(t *testing.T) {
		c := newStyleMixedTestCommando(t)
		// RegionId maps to biz-region-id in metadata, but the engine's Known
		// list does not contain it: do not suggest what the engine rejects.
		cause := unknownFlagUsageError("RegionId", "instance-id")
		err := c.adaptStyleMixedFlagError(cause, args, cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer)))
		assert.Equal(t, cause, err)
	})

	t.Run("profile case error is left to its dedicated adapter", func(t *testing.T) {
		c := newStyleMixedTestCommando(t)
		cause := &kebabProfileFlagCaseError{cause: unknownFlagUsageError("Profile"), product: "ecs", command: "describe-instances"}
		err := c.adaptStyleMixedFlagError(cause, args, cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer)))
		assert.Equal(t, cause, err)
	})
}

func TestAdaptStyleMixedFlagErrorAgentEnvelope(t *testing.T) {
	mixed := &styleMixedFlagError{
		cause:      unknownFlagUsageError("RegionId", "biz-region-id"),
		product:    "ecs",
		command:    "describe-instances",
		flag:       "RegionId",
		suggestion: "--biz-region-id",
		equivalent: "aliyun ecs DescribeInstances --RegionId cn-hangzhou",
	}
	envelope := requireAgentEnvelope(t, mixed, []string{"ecs", "describe-instances"}, nil)
	assert.Equal(t, `unknown flag --RegionId`, envelope.Message)
	assert.Equal(t, []string{"--biz-region-id"}, envelope.DidYouMean)
	assert.Equal(t, "switch_command_style", envelope.Recovery.Action)
	assert.Equal(t, "aliyun ecs DescribeInstances --RegionId cn-hangzhou", envelope.Recovery.Command)
}

func TestInvalidParameterError_AgentEnvelopeWithEquivalentCommand(t *testing.T) {
	err := &InvalidParameterError{
		Name:              "biz-region-id",
		ProductCode:       "ecs",
		ApiName:           "DescribeInstances",
		kebabToRaw:        map[string]string{"biz-region-id": "RegionId"},
		equivalentCommand: "aliyun ecs describe-instances --biz-region-id cn-hangzhou",
	}
	envelope := requireAgentEnvelope(t, err, []string{"ecs", "DescribeInstances"}, nil)
	assert.Equal(t, `"--biz-region-id" is not a valid parameter or flag.`, envelope.Message)
	assert.Equal(t, []string{"--RegionId"}, envelope.DidYouMean)
	assert.Equal(t, "switch_command_style", envelope.Recovery.Action)
	assert.Equal(t, "aliyun ecs describe-instances --biz-region-id cn-hangzhou", envelope.Recovery.Command)
}

func TestInvalidParameterError_AgentEnvelopeWithoutEquivalentKeepsSearchRecovery(t *testing.T) {
	err := &InvalidParameterError{
		Name:           "InstnaceId",
		ProductCode:    "ecs",
		ApiName:        "DescribeInstances",
		ParameterNames: []string{"InstanceId"},
	}
	envelope := requireAgentEnvelope(t, err, []string{"ecs", "DescribeInstances"}, nil)
	assert.Equal(t, []string{"--InstanceId"}, envelope.DidYouMean)
	assert.NotEqual(t, "switch_command_style", envelope.Recovery.Action)
}

func TestRegionConfusionNote(t *testing.T) {
	// fires only when --region was passed AND a missing parameter is region-ish
	assert.NotEmpty(t, regionConfusionNote(true, []string{"--biz-region-id"}))
	assert.NotEmpty(t, regionConfusionNote(true, []string{"--region-id"}))
	assert.Empty(t, regionConfusionNote(false, []string{"--biz-region-id"}))
	assert.Empty(t, regionConfusionNote(true, []string{"--instance-id"}))
}

func TestAnnotateRegionConfusion(t *testing.T) {
	cause := &engine.UsageError{
		Code: "MISSING_REQUIRED_PARAMETER",
		Err:  &runtime.MissingRequiredError{Flags: []string{"--biz-region-id"}},
	}
	ctxWithRegion := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	ctxWithRegion.Flags().Add(newAssignedFlag("region", "cn-hangzhou"))

	err := annotateRegionConfusion(cause, ctxWithRegion)
	var wrapped *regionConfusionError
	require.ErrorAs(t, err, &wrapped)
	assert.Contains(t, wrapped.Error(), "missing required parameter(s): --biz-region-id")
	assert.Contains(t, wrapped.Error(), "--region only selects the region for signing and endpoint resolution")

	ctxNoRegion := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	same := annotateRegionConfusion(cause, ctxNoRegion)
	assert.Equal(t, cause, same)
}

func TestRegionConfusionAgentHint(t *testing.T) {
	cause := &runtime.MissingRequiredError{Flags: []string{"--biz-region-id"}}
	note := regionConfusionNote(true, cause.Flags)
	wrapped := &regionConfusionError{cause: cause, note: note}
	envelope := requireAgentEnvelope(t, wrapped, []string{"ecs", "describe-instances"}, nil)
	assert.Equal(t, "missing required parameter(s): --biz-region-id", envelope.Message)
	assert.Contains(t, envelope.Recovery.Hint, "--region only selects the region")

	noNote := requireAgentEnvelope(t, cause, []string{"ecs", "describe-instances"}, nil)
	assert.NotContains(t, noNote.Recovery.Hint, "--region only selects the region")
}
