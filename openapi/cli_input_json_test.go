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
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/meta"
)

func writeCliInputJSON(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "params.json")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func newCliInputJSONContext(t *testing.T) *cli.Context {
	t.Helper()
	root := &cli.Command{Name: "aliyun", EnableUnknownFlag: true}
	config.AddFlags(root.Flags())
	AddFlags(root.Flags())
	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	ctx.EnterCommand(root)
	return ctx
}

func assignCliInputJSON(t *testing.T, ctx *cli.Context, path string) {
	t.Helper()
	flag := CliInputJSONFlag(ctx.Flags())
	require.NotNil(t, flag)
	flag.SetValue(path)
	flag.SetAssigned(true)
}

func TestParseCliInputJSONFileScalars(t *testing.T) {
	path := writeCliInputJSON(t, `{
		"InstanceName": "web-1",
		"Amount": 9007199254740993,
		"Ratio": -0.5,
		"DryRun": true,
		"Tag.1.Key": "env",
		"Quoted": "a\"b\nc"
	}`)
	pairs, err := parseCliInputJSONFile(path)
	require.NoError(t, err)
	assert.Equal(t, []cliInputJSONPair{
		{Name: "InstanceName", Value: "web-1"},
		{Name: "Amount", Value: "9007199254740993"},
		{Name: "Ratio", Value: "-0.5"},
		{Name: "DryRun", Value: "true"},
		{Name: "Tag.1.Key", Value: "env"},
		{Name: "Quoted", Value: "a\"b\nc"},
	}, pairs)
}

func TestParseCliInputJSONFileDuplicateKeyKeepsLast(t *testing.T) {
	path := writeCliInputJSON(t, `{"InstanceName":"first","InstanceName":"second"}`)
	pairs, err := parseCliInputJSONFile(path)
	require.NoError(t, err)
	assert.Equal(t, []cliInputJSONPair{{Name: "InstanceName", Value: "second"}}, pairs)
}

func TestParseCliInputJSONFileEmptyObject(t *testing.T) {
	path := writeCliInputJSON(t, `{}`)
	pairs, err := parseCliInputJSONFile(path)
	require.NoError(t, err)
	assert.Empty(t, pairs)
}

func TestParseCliInputJSONFileErrors(t *testing.T) {
	for _, tc := range []struct {
		name    string
		content string
		want    string
	}{
		{"root array", `[{"A":1}]`, "root must be a JSON object"},
		{"root string", `"A"`, "root must be a JSON object"},
		{"invalid json", `{"A":`, "invalid JSON"},
		{"trailing content", `{"A":"1"} {"B":"2"}`, "unexpected trailing content"},
		{"null value", `{"A":null}`, "null values are not supported"},
		{"array value", `{"A":[1]}`, "only string, number, and boolean values are supported"},
		{"object value", `{"A":{"B":1}}`, "only string, number, and boolean values are supported"},
		{"empty key", `{"":"1"}`, "parameter names must not be empty"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseCliInputJSONFile(writeCliInputJSON(t, tc.content))
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
			assert.Contains(t, err.Error(), "--cli-input-json")
		})
	}
}

func TestParseCliInputJSONFileUnreadable(t *testing.T) {
	_, err := parseCliInputJSONFile(filepath.Join(t.TempDir(), "missing.json"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot read parameters file")
	assert.Contains(t, err.Error(), "missing.json")

	_, err = parseCliInputJSONFile(t.TempDir())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--cli-input-json")
}

func TestInjectCliInputJSONParamsFillsUnassigned(t *testing.T) {
	ctx := newCliInputJSONContext(t)
	path := writeCliInputJSON(t, `{"InstanceName":"from-file","Tag.1.Key":"env","DryRun":true}`)
	assignCliInputJSON(t, ctx, path)

	require.NoError(t, injectCliInputJSONParams(ctx))

	for name, want := range map[string]string{"InstanceName": "from-file", "Tag.1.Key": "env", "DryRun": "true"} {
		flag := ctx.UnknownFlags().Get(name)
		require.NotNil(t, flag, "parameter %s must be injected", name)
		got, ok := flag.GetValue()
		assert.True(t, ok)
		assert.Equal(t, want, got)
	}
}

func TestInjectCliInputJSONParamsExplicitWins(t *testing.T) {
	ctx := newCliInputJSONContext(t)
	explicit, err := ctx.UnknownFlags().AddByName("InstanceName")
	require.NoError(t, err)
	explicit.SetValue("from-cli")
	explicit.SetAssigned(true)

	assignCliInputJSON(t, ctx, writeCliInputJSON(t, `{"InstanceName":"from-file","ZoneId":"cn-hangzhou-a"}`))
	require.NoError(t, injectCliInputJSONParams(ctx))

	got, _ := ctx.UnknownFlags().Get("InstanceName").GetValue()
	assert.Equal(t, "from-cli", got)
	got, _ = ctx.UnknownFlags().Get("ZoneId").GetValue()
	assert.Equal(t, "cn-hangzhou-a", got)
}

func TestInjectCliInputJSONParamsNotAssigned(t *testing.T) {
	ctx := newCliInputJSONContext(t)
	require.NoError(t, injectCliInputJSONParams(ctx))
	assert.Empty(t, ctx.UnknownFlags().Flags())
}

func TestInjectCliInputJSONParamsRegionIDUsesRootFlag(t *testing.T) {
	ctx := newCliInputJSONContext(t)
	assignCliInputJSON(t, ctx, writeCliInputJSON(t, `{"RegionId":"cn-beijing","DeploymentSetName":"Default"}`))

	require.NoError(t, injectCliInputJSONParams(ctx))

	region := config.RegionIdFlag(ctx.Flags())
	require.NotNil(t, region)
	assert.True(t, region.IsAssigned())
	got, _ := region.GetValue()
	assert.Equal(t, "cn-beijing", got)
	assert.Nil(t, ctx.UnknownFlags().Get("RegionId"), "RegionId is applied through the root flag, not as a dynamic parameter")
	assert.NotNil(t, ctx.UnknownFlags().Get("DeploymentSetName"))
}

func TestInjectCliInputJSONParamsRegionIDExplicitWins(t *testing.T) {
	ctx := newCliInputJSONContext(t)
	region := config.RegionIdFlag(ctx.Flags())
	require.NotNil(t, region)
	region.SetValue("cn-shanghai")
	region.SetAssigned(true)

	assignCliInputJSON(t, ctx, writeCliInputJSON(t, `{"RegionId":"cn-beijing"}`))
	require.NoError(t, injectCliInputJSONParams(ctx))

	got, _ := region.GetValue()
	assert.Equal(t, "cn-shanghai", got)
}

func TestInjectCliInputJSONParamsRegionIDFileAliasKeepsExplicitValue(t *testing.T) {
	ctx := newCliInputJSONContext(t)
	fileAlias, err := ctx.UnknownFlags().AddByName(config.RegionIdFlagName + "-FILE")
	require.NoError(t, err)
	fileAlias.SetValue("cn-hangzhou")
	fileAlias.SetAssigned(true)

	assignCliInputJSON(t, ctx, writeCliInputJSON(t, `{"RegionId":"cn-beijing","DeploymentSetName":"Default"}`))
	require.NoError(t, injectCliInputJSONParams(ctx))

	region := config.RegionIdFlag(ctx.Flags())
	require.NotNil(t, region)
	assert.False(t, region.IsAssigned(), "an explicit --RegionId-FILE value must win over the file")
	got, _ := fileAlias.GetValue()
	assert.Equal(t, "cn-hangzhou", got)
	assert.NotNil(t, ctx.UnknownFlags().Get("DeploymentSetName"))

	// CheckApiParamWithBuildInArgs copies an assigned root RegionId into the
	// parameter flags; with the -FILE alias registered that used to panic.
	v1 := []canonicalmeta.V1Parameter{{Name: "RegionId", Position: "query"}}
	c, _, _ := newTestCommando()
	assert.NotPanics(t, func() {
		c.CheckApiParamWithBuildInArgs(ctx, &canonicalmeta.API{V1Parameters: &v1})
	})
}

func TestCliInputJSONKeyValidation(t *testing.T) {
	ctx := newCliInputJSONContext(t)

	_, err := readCliInputJSONPairsOfFile(t, ctx, `{"InstanceName-FILE":"x"}`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "-FILE")

	_, err = readCliInputJSONPairsOfFile(t, ctx, `{"region":"cn-hangzhou"}`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "conflicts with the global flag --region")

	_, err = readCliInputJSONPairsOfFile(t, ctx, `{"RegionId":"cn-beijing"}`)
	require.NoError(t, err, "RegionId is the standard API parameter and behaves like --RegionId")
}

func readCliInputJSONPairsOfFile(t *testing.T, ctx *cli.Context, content string) ([]cliInputJSONPair, error) {
	t.Helper()
	assignCliInputJSON(t, ctx, writeCliInputJSON(t, content))
	return readCliInputJSONPairs(ctx, true)
}

func TestCliInputJSONKeyShapeValidation(t *testing.T) {
	for _, key := range []string{"-x", "a=b", "a b", "-"} {
		t.Run(key, func(t *testing.T) {
			ctx := newCliInputJSONContext(t)
			_, err := readCliInputJSONPairsOfFile(t, ctx, `{"`+key+`":"1"}`)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "parameter names must start with a letter or digit")
			assert.Contains(t, err.Error(), key)
		})
	}

	ctx := newCliInputJSONContext(t)
	_, err := readCliInputJSONPairsOfFile(t, ctx, `{"Tag.1.Key":"v","deployment-set-name":"x","Under_score":"y"}`)
	require.NoError(t, err)
}

func TestCliInputJSONRegionIDRouteValidation(t *testing.T) {
	ctx := newCliInputJSONContext(t)
	path := writeCliInputJSON(t, `{"RegionId":"cn-beijing"}`)
	assignCliInputJSON(t, ctx, path)

	pairs, err := readCliInputJSONPairs(ctx, true)
	require.NoError(t, err, "the legacy route keeps applying RegionId like --RegionId")
	require.Len(t, pairs, 1)

	_, err = readCliInputJSONPairs(ctx, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "RegionId is only supported by legacy PascalCase commands")
	assert.Contains(t, err.Error(), path)
}

func TestCliInputJSONKeyValidationRejectsEngineReservedFlag(t *testing.T) {
	for _, key := range []string{"all-pages", "pager", "no-stream", "waiter"} {
		t.Run(key, func(t *testing.T) {
			ctx := newCliInputJSONContext(t)
			path := writeCliInputJSON(t, `{"`+key+`":"path=Instances.Instance"}`)
			assignCliInputJSON(t, ctx, path)

			_, err := readCliInputJSONPairs(ctx, false)
			require.Error(t, err, "hidden engine flags must not be imported from the file")
			assert.Contains(t, err.Error(), "conflicts with the global flag --"+key)
			assert.Contains(t, err.Error(), path)

			_, err = expandCliInputJSONArgv(ctx, []string{"fakeops", "do", "--cli-input-json", path})
			require.Error(t, err)
		})
	}
}

func TestCliInputJSONKeyValidationKeepsApiParametersOnEngineRoute(t *testing.T) {
	ctx := newCliInputJSONContext(t)
	assignCliInputJSON(t, ctx, writeCliInputJSON(t, `{"deployment-set-name":"ds","biz-region-id":"cn-hangzhou"}`))

	pairs, err := readCliInputJSONPairs(ctx, false)
	require.NoError(t, err)
	assert.Equal(t, []cliInputJSONPair{
		{Name: "deployment-set-name", Value: "ds"},
		{Name: "biz-region-id", Value: "cn-hangzhou"},
	}, pairs)
}

func TestCliInputJSONKeyValidationLegacyRouteSkipsEngineRegistry(t *testing.T) {
	ctx := newCliInputJSONContext(t)
	// Legacy V1 SDK commands validate parameter names against the API
	// definition, so the engine registry only applies to the argv expansion.
	_, err := readCliInputJSONPairsOfFile(t, ctx, `{"Pager":"legacy-param"}`)
	require.NoError(t, err)
}

func TestExpandCliInputJSONArgvStripsFlagForEmptyObject(t *testing.T) {
	ctx := newCliInputJSONContext(t)
	assignCliInputJSON(t, ctx, writeCliInputJSON(t, `{}`))

	got, err := expandCliInputJSONArgv(ctx, []string{"fakeops", "do", "--cli-input-json", "params.json"})
	require.NoError(t, err)
	assert.Equal(t, []string{"fakeops", "do"}, got)

	got, err = expandCliInputJSONArgv(ctx, []string{"fakeops", "do", "--cli-input-json=params.json", "tail"})
	require.NoError(t, err)
	assert.Equal(t, []string{"fakeops", "do", "tail"}, got)
}

func TestExpandCliInputJSONArgvRejectsRegionIDKey(t *testing.T) {
	ctx := newCliInputJSONContext(t)
	assignCliInputJSON(t, ctx, writeCliInputJSON(t, `{"RegionId":"cn-hangzhou"}`))

	_, err := expandCliInputJSONArgv(ctx, []string{"ecs", "create-deployment-set", "--cli-input-json", "params.json"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `parameter "RegionId"`)
	assert.Contains(t, err.Error(), "biz-region-id")
}

func TestExpandCliInputJSONArgvRejectsInvalidKeyShape(t *testing.T) {
	ctx := newCliInputJSONContext(t)
	assignCliInputJSON(t, ctx, writeCliInputJSON(t, `{"-x":"1"}`))

	_, err := expandCliInputJSONArgv(ctx, []string{"fakeops", "do", "--cli-input-json", "params.json"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parameter names must start with a letter or digit")
	assert.NotContains(t, err.Error(), "---x")
}

func TestExpandCliInputJSONArgvInPlace(t *testing.T) {
	ctx := newCliInputJSONContext(t)
	assignCliInputJSON(t, ctx, writeCliInputJSON(t, `{"amount":2,"dry-run":true}`))

	got, err := expandCliInputJSONArgv(ctx, []string{"ecs", "create-deployment-set", "--cli-input-json", "params.json", "extra"})
	require.NoError(t, err)
	assert.Equal(t, []string{"ecs", "create-deployment-set", "--amount", "2", "--dry-run", "true", "extra"}, got)
}

func TestExpandCliInputJSONArgvEqualsForm(t *testing.T) {
	ctx := newCliInputJSONContext(t)
	assignCliInputJSON(t, ctx, writeCliInputJSON(t, `{"amount":2}`))

	got, err := expandCliInputJSONArgv(ctx, []string{"ecs", "create", "--cli-input-json=params.json"})
	require.NoError(t, err)
	assert.Equal(t, []string{"ecs", "create", "--amount", "2"}, got)
}

func TestExpandCliInputJSONArgvExplicitParameterWins(t *testing.T) {
	ctx := newCliInputJSONContext(t)
	assignCliInputJSON(t, ctx, writeCliInputJSON(t, `{"amount":2,"name":"file"}`))

	got, err := expandCliInputJSONArgv(ctx, []string{"ecs", "create", "--amount", "9", "--cli-input-json", "params.json"})
	require.NoError(t, err)
	assert.Equal(t, []string{"ecs", "create", "--amount", "9", "--name", "file"}, got)
}

func TestExpandCliInputJSONArgvColonForm(t *testing.T) {
	ctx := newCliInputJSONContext(t)
	assignCliInputJSON(t, ctx, writeCliInputJSON(t, `{"amount":2}`))

	got, err := expandCliInputJSONArgv(ctx, []string{"ecs", "create-deployment-set", "--cli-input-json:params.json"})
	require.NoError(t, err)
	assert.Equal(t, []string{"ecs", "create-deployment-set", "--amount", "2"}, got)

	empty := newCliInputJSONContext(t)
	assignCliInputJSON(t, empty, writeCliInputJSON(t, `{}`))
	got, err = expandCliInputJSONArgv(empty, []string{"fakeops", "do", "--cli-input-json:params.json"})
	require.NoError(t, err)
	assert.Equal(t, []string{"fakeops", "do"}, got)
}

func TestExpandCliInputJSONArgvColonExplicitParameterWins(t *testing.T) {
	ctx := newCliInputJSONContext(t)
	assignCliInputJSON(t, ctx, writeCliInputJSON(t, `{"deployment-set-name":"file","amount":2}`))

	got, err := expandCliInputJSONArgv(ctx, []string{"ecs", "create", "--deployment-set-name:cli", "--cli-input-json", "params.json"})
	require.NoError(t, err)
	assert.Equal(t, []string{"ecs", "create", "--deployment-set-name:cli", "--amount", "2"}, got)
}

func TestExpandCliInputJSONArgvInsertedBeforeTerminator(t *testing.T) {
	ctx := newCliInputJSONContext(t)
	assignCliInputJSON(t, ctx, writeCliInputJSON(t, `{"name":"file"}`))

	got, err := expandCliInputJSONArgv(ctx, []string{"ecs", "create", "--", "--positional"})
	require.NoError(t, err)
	assert.Equal(t, []string{"ecs", "create", "--name", "file", "--", "--positional"}, got)
}

func TestExpandCliInputJSONArgvLeavesTailAfterTerminator(t *testing.T) {
	ctx := newCliInputJSONContext(t)
	assignCliInputJSON(t, ctx, writeCliInputJSON(t, `{"name":"file"}`))

	got, err := expandCliInputJSONArgv(ctx, []string{"ecs", "create", "--", "--cli-input-json", "params.json"})
	require.NoError(t, err)
	assert.Equal(t, []string{"ecs", "create", "--name", "file", "--", "--cli-input-json", "params.json"}, got)
}

func TestExpandCliInputJSONArgvNotAssigned(t *testing.T) {
	ctx := newCliInputJSONContext(t)
	argv := []string{"ecs", "create", "--name", "x"}
	got, err := expandCliInputJSONArgv(ctx, argv)
	require.NoError(t, err)
	assert.Equal(t, argv, got)
}

func TestRejectCliInputJSONLocalRoute(t *testing.T) {
	root := &cli.Command{Name: "aliyun", EnableUnknownFlag: true}
	root.AddSubCommand(&cli.Command{Name: "configure"})
	root.AddSubCommand(&cli.Command{Name: "version"})
	AddFlags(root.Flags())
	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	ctx.EnterCommand(root)

	t.Run("local subcommand rejected", func(t *testing.T) {
		err := rejectCliInputJSONLocalRoute(ctx, []string{"configure", "--cli-input-json", "params.json"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "only applies to product API commands")
		assert.Contains(t, err.Error(), "configure")
	})

	t.Run("version rejected", func(t *testing.T) {
		err := rejectCliInputJSONLocalRoute(ctx, []string{"version", "--cli-input-json=params.json"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), `"version"`)
	})

	t.Run("product command allowed", func(t *testing.T) {
		assert.NoError(t, rejectCliInputJSONLocalRoute(ctx, []string{"ecs", "CreateDeploymentSet", "--cli-input-json", "params.json"}))
	})

	t.Run("help request allowed", func(t *testing.T) {
		assert.NoError(t, rejectCliInputJSONLocalRoute(ctx, []string{"configure", "--help", "--cli-input-json", "params.json"}))
	})

	t.Run("help command rejected", func(t *testing.T) {
		err := rejectCliInputJSONLocalRoute(ctx, []string{"help", "ecs", "--cli-input-json", "params.json"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "only applies to product API commands")
	})

	t.Run("flag after terminator ignored", func(t *testing.T) {
		assert.NoError(t, rejectCliInputJSONLocalRoute(ctx, []string{"configure", "--", "--cli-input-json", "params.json"}))
	})

	t.Run("inline value flag before target", func(t *testing.T) {
		err := rejectCliInputJSONLocalRoute(ctx, []string{"--profile=default", "configure", "--cli-input-json", "params.json"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "configure")
	})

	t.Run("flag absent", func(t *testing.T) {
		assert.NoError(t, rejectCliInputJSONLocalRoute(ctx, []string{"configure"}))
	})
}

func TestRejectCliInputJSONVersionRoute(t *testing.T) {
	ctx := newCliInputJSONContext(t)
	assignCliInputJSON(t, ctx, writeCliInputJSON(t, `{"name":"file"}`))

	err := rejectCliInputJSONVersionRoute(ctx, "version")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `only applies to product API commands, not "version"`)

	assert.NoError(t, rejectCliInputJSONVersionRoute(ctx, "create-deployment-set"))

	unassigned := newCliInputJSONContext(t)
	assert.NoError(t, rejectCliInputJSONVersionRoute(unassigned, "version"))
}

func TestCommandoMainRequiresProductForCliInputJSON(t *testing.T) {
	c, _, _ := newTestCommando()
	ctx := newCliInputJSONContext(t)
	assignCliInputJSON(t, ctx, writeCliInputJSON(t, `{"InstanceName":"web-1"}`))

	err := c.main(ctx, []string{"ecs"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--cli-input-json requires a product and an API name")
}

// newCliInputJSONInvokeContext mirrors the root flag state the legacy invoker
// needs for a dry-run, without touching any real credential or endpoint.
func newCliInputJSONInvokeContext(t *testing.T, stdout, stderr *bytes.Buffer) *cli.Context {
	t.Helper()
	ctx := cli.NewCommandContext(stdout, stderr)
	cmd := &cli.Command{}
	cmd.EnableUnknownFlag = true
	AddFlags(cmd.Flags())
	ctx.EnterCommand(cmd)

	skip := config.NewSkipSecureVerify()
	skip.SetAssigned(true)
	ctx.Flags().Add(skip)

	region := config.NewRegionFlag()
	region.SetAssigned(true)
	region.SetValue("cn-hangzhou")
	ctx.Flags().Add(region)

	endpoint := config.NewEndpointFlag()
	endpoint.SetAssigned(true)
	endpoint.SetValue("ecs.cn-hangzhou.aliyuncs.com")
	ctx.Flags().Add(endpoint)

	ctx.Flags().Add(config.NewRegionIdFlag())

	VersionFlag(ctx.Flags()).SetAssigned(true)
	VersionFlag(ctx.Flags()).SetValue("2014-05-26")
	ForceFlag(ctx.Flags()).SetAssigned(true)
	DryRunJsonFlag(ctx.Flags()).SetAssigned(true)
	return ctx
}

func TestCliInputJSONLegacyDryRunMatchesExplicitFlags(t *testing.T) {
	profile := config.Profile{
		Language:        "en",
		Mode:            "AK",
		AccessKeyId:     "accesskeyid",
		AccessKeySecret: "accesskeysecret",
		RegionId:        "cn-hangzhou",
	}
	invoke := func(t *testing.T, ctx *cli.Context) CliDryRunOutput {
		t.Helper()
		stdout := ctx.Stdout().(*bytes.Buffer)
		command := NewCommando(stdout, profile)
		require.NoError(t, command.processInvoke(ctx, "ecs", "CreateDeploymentSet", ""))
		var out CliDryRunOutput
		require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(stdout.String())), &out))
		return out
	}

	fromFile := newCliInputJSONInvokeContext(t, new(bytes.Buffer), new(bytes.Buffer))
	assignCliInputJSON(t, fromFile, writeCliInputJSON(t, `{
		"RegionId": "cn-hangzhou",
		"DeploymentSetName": "Default",
		"Description": "My default deployment set.",
		"Domain": "Default",
		"Granularity": "Host",
		"Strategy": "LooseDispersion"
	}`))
	require.NoError(t, injectCliInputJSONParams(fromFile))

	explicit := newCliInputJSONInvokeContext(t, new(bytes.Buffer), new(bytes.Buffer))
	region := config.RegionIdFlag(explicit.Flags())
	region.SetAssigned(true)
	region.SetValue("cn-hangzhou")
	for name, value := range map[string]string{
		"DeploymentSetName": "Default",
		"Description":       "My default deployment set.",
		"Domain":            "Default",
		"Granularity":       "Host",
		"Strategy":          "LooseDispersion",
	} {
		flag, err := explicit.UnknownFlags().AddByName(name)
		require.NoError(t, err)
		flag.SetValue(value)
		flag.SetAssigned(true)
	}

	assert.Equal(t, invoke(t, explicit).Query, invoke(t, fromFile).Query)
}

func TestCliInputJSONLegacyRestfulDryRunUsesFileParameters(t *testing.T) {
	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	ctx := newCliInputJSONInvokeContext(t, stdout, stderr)
	endpoint := config.EndpointFlag(ctx.Flags())
	require.NotNil(t, endpoint)
	endpoint.SetValue("cs.cn-hangzhou.aliyuncs.com")
	VersionFlag(ctx.Flags()).SetValue("2015-12-15")

	assignCliInputJSON(t, ctx, writeCliInputJSON(t, `{"name":"demo","size":10}`))
	require.NoError(t, injectCliInputJSONParams(ctx))

	profile := config.Profile{
		Language:        "en",
		Mode:            "AK",
		AccessKeyId:     "accesskeyid",
		AccessKeySecret: "accesskeysecret",
		RegionId:        "cn-hangzhou",
	}
	command := NewCommando(stdout, profile)
	repo, err := meta.MockLoadRepository([]meta.Product{
		{Code: "cs", Version: "2015-12-15", ApiStyle: "restful"},
	})
	require.NoError(t, err)
	command.library.builtinRepo = repo

	require.NoError(t, command.processInvoke(ctx, "cs", "GET", "/clusters"))

	var out CliDryRunOutput
	require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(stdout.String())), &out))
	assert.Equal(t, "ROA", out.Style)
	assert.Equal(t, "demo", out.Query["name"])
	assert.Equal(t, "10", out.Query["size"])
}

func TestCliInputJSONErrorUnwrap(t *testing.T) {
	cause := errors.New("boom")
	err := &cliInputJSONError{Path: "p.json", Key: "Tag.1.Key", Reason: "bad", Err: cause}
	assert.True(t, errors.Is(err, cause))
	assert.Contains(t, err.Error(), `--cli-input-json: p.json: parameter "Tag.1.Key": bad: boom`)
}

func TestReadCliInputJSONPairsEmptyPath(t *testing.T) {
	ctx := newCliInputJSONContext(t)
	flag := CliInputJSONFlag(ctx.Flags())
	flag.SetValue("   ")
	flag.SetAssigned(true)

	_, err := readCliInputJSONPairs(ctx, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "file path must not be empty")
}

func TestReadCliInputJSONPairsPropagatesParseError(t *testing.T) {
	ctx := newCliInputJSONContext(t)
	assignCliInputJSON(t, ctx, filepath.Join(t.TempDir(), "absent.json"))

	_, err := readCliInputJSONPairs(ctx, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot read parameters file")
}

func TestParseCliInputJSONFileTokenErrors(t *testing.T) {
	for name, content := range map[string]string{
		"empty file":           "",
		"trailing comma":       `{"a":1,}`,
		"unterminated object":  `{"a":1`,
		"missing value":        `{"a":}`,
		"key without colon":    `{"a" 1}`,
		"nested unclosed body": `{"a":{"b":1}`,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := parseCliInputJSONFile(writeCliInputJSON(t, content))
			require.Error(t, err)
			assert.Contains(t, err.Error(), "--cli-input-json")
		})
	}
}

func TestInjectCliInputJSONParamsRequiresUnknownFlags(t *testing.T) {
	root := &cli.Command{Name: "aliyun"}
	config.AddFlags(root.Flags())
	AddFlags(root.Flags())
	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	ctx.EnterCommand(root)
	require.Nil(t, ctx.UnknownFlags())

	assignCliInputJSON(t, ctx, writeCliInputJSON(t, `{"InstanceName":"web-1"}`))
	err := injectCliInputJSONParams(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parameter flags are unavailable")
}

func TestExpandCliInputJSONArgvAppendsWhenFlagAbsent(t *testing.T) {
	ctx := newCliInputJSONContext(t)
	assignCliInputJSON(t, ctx, writeCliInputJSON(t, `{"Amount":2}`))

	got, err := expandCliInputJSONArgv(ctx, []string{"ecs", "create-deployment-set", "extra"})
	require.NoError(t, err)
	assert.Equal(t, []string{"ecs", "create-deployment-set", "extra", "--Amount", "2"}, got)
}

func TestRejectCliInputJSONLocalRouteGuards(t *testing.T) {
	assert.NoError(t, rejectCliInputJSONLocalRoute(nil, []string{"configure", "--cli-input-json", "p.json"}))

	root := &cli.Command{Name: "aliyun", EnableUnknownFlag: true}
	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	ctx.EnterCommand(root)
	assert.NoError(t, rejectCliInputJSONLocalRoute(ctx, []string{"configure", "--cli-input-json", "p.json"}))

	full := newCliInputJSONContext(t)
	assert.NoError(t, rejectCliInputJSONLocalRoute(full, []string{"--cli-input-json", "p.json"}))
}

func TestCliInputJSONFlagTakesValue(t *testing.T) {
	flags := NewFlagSetForCliInputJSON()
	for _, tc := range []struct {
		arg  string
		want bool
	}{
		{"--cli-input-json", true},
		{"--yes", false},
		{"--region", true},
		{"--region=cn-hangzhou", false},
		{"-Z", true},
		{"-y", false},
		{"-", false},
	} {
		assert.Equal(t, tc.want, cliInputJSONFlagTakesValue(flags, tc.arg), tc.arg)
	}
	assert.True(t, cliInputJSONFlagTakesValue(nil, "--anything"))
}

func NewFlagSetForCliInputJSON() *cli.FlagSet {
	fs := cli.NewFlagSet()
	config.AddFlags(fs)
	AddFlags(fs)
	return fs
}

func TestCliInputJSONFlagPresentSkipsOnlyParserValues(t *testing.T) {
	flags := NewFlagSetForCliInputJSON()
	assert.True(t, cliInputJSONFlagPresent([]string{"oss", "ls", "oss://bucket", "--recursive", "--cli-input-json", "p.json"}, flags),
		"an unknown flag never consumes a following --xxx token")
	assert.True(t, cliInputJSONFlagPresent([]string{"--region", "--cli-input-json", "p.json"}, flags),
		"a known flag never consumes a following --xxx token")
	assert.True(t, cliInputJSONFlagPresent([]string{"--recursive", "value", "--cli-input-json:p.json"}, flags))
	assert.False(t, cliInputJSONFlagPresent([]string{"--region", "-1", "--yes"}, flags),
		"a known long flag consumes a dash-leading value")
	assert.Equal(t, "configure", cliInputJSONFirstPositional([]string{"--recursive", "value", "configure"}, flags))
	assert.Equal(t, "", cliInputJSONFirstPositional([]string{"--recursive", "--region", "cn-hangzhou"}, flags))
	assert.False(t, cliInputJSONConsumesNext(flags, "-x", "-1"), "only known long flags consume dash-leading values")
	assert.False(t, cliInputJSONConsumesNext(nil, "--region", "-1"))
	assert.True(t, cliInputJSONConsumesNext(nil, "--region", "cn-hangzhou"))
}

func TestCliInputJSONFirstPositionalBranches(t *testing.T) {
	flags := NewFlagSetForCliInputJSON()
	assert.Empty(t, cliInputJSONFirstPositional([]string{"--", "ecs"}, flags))
	assert.Empty(t, cliInputJSONFirstPositional([]string{"--yes"}, flags))
	assert.Empty(t, cliInputJSONFirstPositional(nil, flags))
	assert.Equal(t, "ecs", cliInputJSONFirstPositional([]string{"--region", "cn-hangzhou", "ecs"}, flags))
}

func TestCommandoMainReportsFileErrorsOnBothRoutes(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "absent.json")
	for name, args := range map[string][]string{
		"kebab":  {"ecs", "describe-instances"},
		"legacy": {"ecs", "DescribeInstances"},
	} {
		t.Run(name, func(t *testing.T) {
			c, _, _ := newTestCommando()
			ctx := newCliInputJSONContext(t)
			assignCliInputJSON(t, ctx, missing)
			err := c.main(ctx, args)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "cannot read parameters file")
		})
	}
}

func TestBeforeParseHelpRouteRejectsLocalCommandWithFile(t *testing.T) {
	c, _, _ := newTestCommando()
	root := &cli.Command{Name: "aliyun", EnableUnknownFlag: true}
	root.AddSubCommand(&cli.Command{Name: "configure"})
	config.AddFlags(root.Flags())
	AddFlags(root.Flags())
	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	ctx.EnterCommand(root)

	handled, err := c.beforeParseHelpRoute(ctx, []string{"configure", "--cli-input-json", "p.json"})
	require.True(t, handled)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "only applies to product API commands")
}
