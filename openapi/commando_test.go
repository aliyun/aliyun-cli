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
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
	"github.com/stretchr/testify/assert"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/cli/plugin"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/aliyun/aliyun-cli/v3/meta"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/safety"
)

// setTestHomeDir sets the test home directory for cross-platform testing.
// Returns a cleanup function that restores the original environment variables.
func setTestHomeDir(t *testing.T, testHome string) func() {
	originalHome := os.Getenv("HOME")
	originalUserProfile := os.Getenv("USERPROFILE")
	originalHomeDrive := os.Getenv("HOMEDRIVE")
	originalHomePath := os.Getenv("HOMEPATH")

	os.Setenv("HOME", testHome)
	if runtime.GOOS == "windows" {
		os.Setenv("USERPROFILE", testHome)
		// Clear HOMEDRIVE and HOMEPATH to ensure USERPROFILE or HOME is used
		os.Unsetenv("HOMEDRIVE")
		os.Unsetenv("HOMEPATH")
	}

	return func() {
		os.Setenv("HOME", originalHome)
		if runtime.GOOS == "windows" {
			os.Setenv("USERPROFILE", originalUserProfile)
			os.Setenv("HOMEDRIVE", originalHomeDrive)
			os.Setenv("HOMEPATH", originalHomePath)
		}
	}
}

func Test_main(t *testing.T) {
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)
	profile := config.Profile{
		Language: "en",
		RegionId: "cn-hangzhou",
	}
	command := NewCommando(w, profile)
	assert.NotNil(t, command)

	cmd := &cli.Command{}
	cmd.EnableUnknownFlag = true
	command.InitWithCommand(cmd)

	AddFlags(cmd.Flags())
	ctx.EnterCommand(cmd)
	ctx.Command().Short = &i18n.Text{}
	err := command.main(ctx, nil)
	assert.Nil(t, err)

	args := []string{"test"}
	profileflag := config.NewProfileFlag()
	configpathflag := config.NewConfigurePathFlag()
	profileflag.SetAssigned(true)
	profileflag.SetValue("ecs")
	skipflag := config.NewSkipSecureVerify()
	skipflag.SetAssigned(true)

	ctx.Flags().Add(profileflag)
	ctx.Flags().Add(skipflag)
	ctx.Flags().Add(config.NewRegionFlag())
	ctx.Flags().Add(configpathflag)

	err = command.main(ctx, args)
	assert.NotNil(t, err)
	assert.Equal(t, "unknown profile ecs, run configure to check", err.Error())
	ctx.Flags().Get("region").SetAssigned(true)
	ctx.Flags().Get("region").SetValue("cn-hangzhou")
	ctx.Flags().Add(config.NewAccessKeyIdFlag())
	ctx.Flags().Get("access-key-id").SetAssigned(true)
	ctx.Flags().Get("access-key-id").SetValue("AccessKeyID")
	ctx.Flags().Add(config.NewAccessKeySecretFlag())
	ctx.Flags().Get("access-key-secret").SetAssigned(true)
	ctx.Flags().Get("access-key-secret").SetValue("AccessKeySecret")
	args = []string{"test"}
	profileflag.SetAssigned(false)
	err = command.main(ctx, args)
	assert.NotNil(t, err)
	assert.Equal(t, "'test' is not a valid command or product. See `aliyun help`.", err.Error())

	ctx.Flags().Get("force").SetAssigned(true)
	ctx.Flags().Get("version").SetAssigned(true)
	ctx.Flags().Get("version").SetValue("2011-11-11")
	args = []string{"ecs", "DescribeRegions"}
	err = command.main(ctx, args)
	assert.NotNil(t, err)
	assert.Equal(t, "unchecked version 2011-11-11", err.Error())

	ctx.Flags().Get("version").SetValue("2016-03-14")
	args = []string{"ecs", "DescribeRegions"}
	err = command.main(ctx, args)
	assert.NotNil(t, err)
	assert.True(t, strings.Contains(err.Error(), "SDK.ServerError\nErrorCode: InvalidAction.NotFound\n"))
	assert.True(t, strings.Contains(err.Error(), "Recommend: https://api.aliyun.com/troubleshoot?q=InvalidAction.NotFound&product=Ecs&requestId="))
	assert.True(t, strings.Contains(err.Error(), "RequestId: "))

	ctx.Flags().Get("force").SetAssigned(false)
	ctx.Flags().Get("version").SetAssigned(false)

	args = []string{"aos", "test2"}
	err = command.main(ctx, args)
	assert.NotNil(t, err)
	assert.Equal(t, "'aos' is not a valid product. See `aliyun help`.", err.Error())

	args = []string{"test", "test2", "test1"}
	err = command.main(ctx, args)
	assert.NotNil(t, err)
	assert.Equal(t, "'test' is not a valid product. See `aliyun help`.", err.Error())

	args = []string{"test", "Test2", "test1", "test3"}
	err = command.main(ctx, args)
	assert.NotNil(t, err)
	assert.Equal(t, "too many arguments", err.Error())

}

func Test_processInvoke(t *testing.T) {
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)
	cmd := &cli.Command{}
	cmd.EnableUnknownFlag = true

	AddFlags(cmd.Flags())
	ctx.EnterCommand(cmd)

	//AddFlags(ctx.Flags())

	skipflag := config.NewSkipSecureVerify()
	skipflag.SetAssigned(true)
	ctx.Flags().Add(skipflag)

	regionflag := config.NewRegionFlag()
	regionflag.SetAssigned(true)
	regionflag.SetValue("cn-hangzhou")
	ctx.Flags().Add(regionflag)

	endpointflag := config.NewEndpointFlag()
	endpointflag.SetAssigned(true)
	endpointflag.SetValue("ecs.cn-hangzhou.aliyuncs")
	ctx.Flags().Add(endpointflag)

	VersionFlag(ctx.Flags()).SetAssigned(true)
	VersionFlag(ctx.Flags()).SetValue("v1.0")

	HeaderFlag(ctx.Flags()).SetValues([]string{"Accept=xml", "Content-Type=json"})

	profile := config.Profile{
		Language:        "en",
		Mode:            "AK",
		AccessKeyId:     "accesskeyid",
		AccessKeySecret: "accesskeysecret",
		RegionId:        "cn-hangzhou",
	}
	command := NewCommando(w, profile)

	productCode := "test"
	apiOrMethod := "get"
	path := "/user"
	ForceFlag(ctx.Flags()).SetAssigned(true)

	err := command.processInvoke(ctx, productCode, apiOrMethod, path)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "lookup ecs.cn-hangzhou.aliyuncs")

	DryRunFlag(ctx.Flags()).SetAssigned(true)
	err = command.processInvoke(ctx, productCode, apiOrMethod, path)
	assert.Nil(t, err)

	DryRunFlag(ctx.Flags()).SetAssigned(false)
	DryRunJsonFlag(ctx.Flags()).SetAssigned(true)
	err = command.processInvoke(ctx, productCode, apiOrMethod, path)
	assert.Nil(t, err)

	DryRunJsonFlag(ctx.Flags()).SetAssigned(false)
	PagerFlag.SetAssigned(true)
	err = command.processInvoke(ctx, productCode, apiOrMethod, path)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "lookup ecs.cn-hangzhou.aliyuncs")

	PagerFlag.SetAssigned(false)
	WaiterFlag.SetAssigned(true)
	err = command.processInvoke(ctx, productCode, apiOrMethod, path)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "lookup ecs.cn-hangzhou.aliyuncs")

	originhookdo := hookdo
	defer func() {
		hookdo = originhookdo
	}()
	hookdo = func(fn func() (*responses.CommonResponse, error)) func() (*responses.CommonResponse, error) {
		resp := responses.NewCommonResponse()
		return func() (*responses.CommonResponse, error) {
			return resp, nil
		}
	}
	WaiterFlag.SetAssigned(false)
	QuietFlag(ctx.Flags()).SetAssigned(true)
	err = command.processInvoke(ctx, productCode, apiOrMethod, path)
	assert.Nil(t, err)

	QuietFlag(ctx.Flags()).SetAssigned(false)
	OutputFlag(ctx.Flags()).SetAssigned(true)
	err = command.processInvoke(ctx, productCode, apiOrMethod, path)
	assert.NotNil(t, err)
	assert.Equal(t, "you need to assign col=col1,col2,... with --output", err.Error())

	OutputFlag(ctx.Flags()).SetAssigned(false)
	err = command.processInvoke(ctx, productCode, apiOrMethod, path)
	assert.Nil(t, err)

	out := `{"requestid":"test","name":"json"}`
	out = sortJSON(out)
	assert.Equal(t, "{\n\t\"name\": \"json\",\n\t\"requestid\": \"test\"\n}", out)

	out = `{"downloadlink":"aaa&bbb"}`
	out = sortJSON(out)
	assert.Equal(t, "{\n\t\"downloadlink\": \"aaa&bbb\"\n}", out)
}

// TestProcessInvoke_DryRunJSON focuses on the --cli-dry-run-json branch of
// processInvoke: a single compact JSON line on stdout with the expected
// product/version/api/region/endpoint fields.
func TestProcessInvoke_DryRunJSON(t *testing.T) {
	newCtx := func(stdout, stderr *bytes.Buffer) *cli.Context {
		ctx := cli.NewCommandContext(stdout, stderr)
		cmd := &cli.Command{}
		cmd.EnableUnknownFlag = true
		AddFlags(cmd.Flags())
		ctx.EnterCommand(cmd)

		skipflag := config.NewSkipSecureVerify()
		skipflag.SetAssigned(true)
		ctx.Flags().Add(skipflag)

		regionflag := config.NewRegionFlag()
		regionflag.SetAssigned(true)
		regionflag.SetValue("cn-hangzhou")
		ctx.Flags().Add(regionflag)

		endpointflag := config.NewEndpointFlag()
		endpointflag.SetAssigned(true)
		endpointflag.SetValue("ecs.cn-hangzhou.aliyuncs.com")
		ctx.Flags().Add(endpointflag)

		VersionFlag(ctx.Flags()).SetAssigned(true)
		VersionFlag(ctx.Flags()).SetValue("2014-05-26")

		ForceFlag(ctx.Flags()).SetAssigned(true)
		DryRunJsonFlag(ctx.Flags()).SetAssigned(true)
		return ctx
	}

	profile := config.Profile{
		Language:        "en",
		Mode:            "AK",
		AccessKeyId:     "accesskeyid",
		AccessKeySecret: "accesskeysecret",
		RegionId:        "cn-hangzhou",
	}

	parseLine := func(t *testing.T, stdout *bytes.Buffer) dryRunInvokeMeta {
		t.Helper()
		line := strings.TrimSpace(stdout.String())
		assert.NotEmpty(t, line, "stdout must not be empty")
		assert.False(t, strings.Contains(line, "\n"), "expected single-line JSON, got: %q", line)
		var m dryRunInvokeMeta
		assert.Nil(t, json.Unmarshal([]byte(line), &m), "stdout must be valid JSON: %q", line)
		return m
	}

	t.Run("ForceRpcInvoker", func(t *testing.T) {
		stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
		ctx := newCtx(stdout, stderr)
		command := NewCommando(stdout, profile)

		err := command.processInvoke(ctx, "ecs", "DescribeRegions", "")
		assert.Nil(t, err)

		m := parseLine(t, stdout)
		// product code comes from library and may be canonicalized (e.g. "Ecs"); compare case-insensitively.
		assert.Equal(t, "ecs", strings.ToLower(m.Product))
		assert.Equal(t, "2014-05-26", m.Version)
		assert.Equal(t, "DescribeRegions", m.API)
		assert.Equal(t, "cn-hangzhou", m.Region)
		assert.Equal(t, "ecs.cn-hangzhou.aliyuncs.com", m.Endpoint)
	})

	t.Run("RestfulInvoker_FallbackToMethodPath", func(t *testing.T) {
		// Unknown product + force on a path → RestfulInvoker without a meta.Api;
		// api field falls back to "<METHOD> <path>".
		stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
		ctx := newCtx(stdout, stderr)
		command := NewCommando(stdout, profile)

		err := command.processInvoke(ctx, "unknown-product-xyz", "GET", "/instances")
		assert.Nil(t, err)

		m := parseLine(t, stdout)
		assert.Equal(t, "unknown-product-xyz", m.Product)
		assert.Equal(t, "2014-05-26", m.Version)
		assert.Equal(t, "GET /instances", m.API)
		assert.Equal(t, "cn-hangzhou", m.Region)
		assert.Equal(t, "ecs.cn-hangzhou.aliyuncs.com", m.Endpoint)
	})
}

func TestProcessInvokeQueryFlag(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)
	cmd := &cli.Command{}
	cmd.EnableUnknownFlag = true
	AddFlags(cmd.Flags())
	ctx.EnterCommand(cmd)

	skipflag := config.NewSkipSecureVerify()
	skipflag.SetAssigned(true)
	ctx.Flags().Add(skipflag)

	regionflag := config.NewRegionFlag()
	regionflag.SetAssigned(true)
	regionflag.SetValue("cn-hangzhou")
	ctx.Flags().Add(regionflag)

	endpointflag := config.NewEndpointFlag()
	endpointflag.SetAssigned(true)
	endpointflag.SetValue("ecs.cn-hangzhou.aliyuncs")
	ctx.Flags().Add(endpointflag)

	VersionFlag(ctx.Flags()).SetAssigned(true)
	VersionFlag(ctx.Flags()).SetValue("v1.0")

	profile := config.Profile{
		Language:        "en",
		Mode:            "AK",
		AccessKeyId:     "accesskeyid",
		AccessKeySecret: "accesskeysecret",
		RegionId:        "cn-hangzhou",
	}
	command := NewCommando(stdout, profile)

	productCode := "test"
	apiOrMethod := "get"
	path := "/user"
	ForceFlag(ctx.Flags()).SetAssigned(true)

	originalHookdo := hookdo
	defer func() {
		hookdo = originalHookdo
	}()

	t.Run("QueryFlagAssignedWithInvalidQuery", func(t *testing.T) {
		PagerFlag.SetAssigned(false)
		hookdo = func(fn func() (*responses.CommonResponse, error)) func() (*responses.CommonResponse, error) {
			return func() (*responses.CommonResponse, error) {
				resp := responses.NewCommonResponse()
				return resp, nil
			}
		}

		QueryFlag(ctx.Flags()).SetAssigned(true)
		QueryFlag(ctx.Flags()).SetValue("invalid[")

		stdout.Reset()
		err := command.processInvoke(ctx, productCode, apiOrMethod, path)
		assert.NoError(t, err)
	})

	t.Run("QueryFlagNotAssigned", func(t *testing.T) {
		hookdo = func(fn func() (*responses.CommonResponse, error)) func() (*responses.CommonResponse, error) {
			return func() (*responses.CommonResponse, error) {
				resp := responses.NewCommonResponse()
				return resp, nil
			}
		}
		QueryFlag(ctx.Flags()).SetAssigned(false)

		stdout.Reset()
		err := command.processInvoke(ctx, productCode, apiOrMethod, path)
		assert.NoError(t, err)
	})
}
func Test_sortJSON(t *testing.T) {
	out := `{"Id":1000000000000000010241024}`
	out = sortJSON(out)
	assert.Equal(t, "{\n\t\"Id\": 1000000000000000010241024\n}", out)
}
func Test_help(t *testing.T) {
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)
	cmd := &cli.Command{}
	cmd.EnableUnknownFlag = true

	AddFlags(cmd.Flags())
	ctx.EnterCommand(cmd)
	ctx.Command().Short = &i18n.Text{}

	profile := config.Profile{
		Language:        "en",
		Mode:            "AK",
		AccessKeyId:     "accesskeyid",
		AccessKeySecret: "accesskeysecret",
		RegionId:        "cn-hangzhou",
	}
	command := NewCommando(w, profile)
	args := []string{}
	err := command.help(ctx, args)
	assert.Nil(t, err)

	args = []string{"test"}
	err = command.help(ctx, args)
	assert.NotNil(t, err)
	assert.Equal(t, "'test' is not a valid product. See `aliyun help`.", err.Error())

	args = []string{"test", "test0"}
	err = command.help(ctx, args)
	assert.NotNil(t, err)
	assert.Equal(t, "'test' is not a valid product. See `aliyun help`.", err.Error())

	args = []string{"test", "test0", "test1"}
	err = command.help(ctx, args)
	assert.NotNil(t, err)
	assert.Equal(t, "too many arguments: 3", err.Error())
}

func Test_complete(t *testing.T) {
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)
	cmd := &cli.Command{}
	cmd.EnableUnknownFlag = true

	AddFlags(cmd.Flags())
	ctx.EnterCommand(cmd)
	ctx.Command().Short = &i18n.Text{}
	ctx.SetCompletion(&cli.Completion{
		Current: "aos",
	})

	profile := config.Profile{
		Language:        "en",
		Mode:            "AK",
		AccessKeyId:     "accesskeyid",
		AccessKeySecret: "accesskeysecret",
		RegionId:        "cn-hangzhou",
	}
	command := NewCommando(w, profile)
	command.library.builtinRepo = meta.LoadRepository()
	args := []string{}
	str := command.complete(ctx, args)
	assert.Equal(t, []string{}, str)

	args = []string{"obs"}
	str = command.complete(ctx, args)
	assert.Equal(t, []string{}, str)

	ctx.SetCompletion(&cli.Completion{
		Current: "DescribeRegions",
	})
	args = []string{"ecs"}
	str = command.complete(ctx, args)
	assert.Equal(t, []string{}, str)

	ctx.SetCompletion(&cli.Completion{
		Current: "DescribeRegions",
	})
	args = []string{"ecs", "aos"}
	str = command.complete(ctx, args)
	assert.Equal(t, []string{}, str)

	args = []string{"aos"}
	str = command.complete(ctx, args)
	assert.Equal(t, []string{}, str)
}

func TestCreateInvoker(t *testing.T) {
	profile := config.NewProfile("test")
	profile.Mode = config.AK
	profile.AccessKeyId = "AccessKeyId"
	profile.AccessKeySecret = "AccessKeySecret"
	profile.RegionId = "cn-hangzhou"
	w := new(bytes.Buffer)
	commando := NewCommando(w, profile)

	tempWriter := new(bytes.Buffer)
	tempStderrWriter := new(bytes.Buffer)
	ctx := cli.NewCommandContext(tempWriter, tempStderrWriter)
	config.AddFlags(ctx.Flags())
	AddFlags(ctx.Flags())
	ctx.Flags().Get("force").SetAssigned(true)
	invoker, err := commando.createInvoker(ctx, "ecs", "DescribeRegions", "")
	rpcinvoker, ok := invoker.(*ForceRpcInvoker)
	assert.True(t, ok)
	assert.Nil(t, err)
	assert.Equal(t, rpcinvoker.method, "DescribeRegions")

	ctx.Flags().Get("version").SetAssigned(true)
	ctx.Flags().Get("version").SetValue("2018-12-01")
	invoker, err = commando.createInvoker(ctx, "cr", "GetRegion", "")
	_, ok = invoker.(*ForceRpcInvoker)
	assert.True(t, ok)
	assert.Nil(t, err)

	ctx.EnterCommand(&cli.Command{})
	ctx.Flags().Add(config.NewRegionFlag())
	ctx.Flags().Add(config.NewRegionIdFlag())
	AddFlags(ctx.Flags())
	ctx.Flags().Get("force").SetAssigned(false)
	ctx.Flags().Get("version").SetAssigned(false)
	invoker, err = commando.createInvoker(ctx, "cs", "Get", "/api/v1/clusters")
	_, ok = invoker.(*RestfulInvoker)
	assert.True(t, ok)
	assert.Nil(t, err)

}

func TestCheckApiParamWithBuildInArgs(t *testing.T) {
	// Initialize cli.Context
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)
	cmd := &cli.Command{}
	cmd.EnableUnknownFlag = true
	ctx.EnterCommand(cmd)

	// Add known flags to context
	knownFlag := &cli.Flag{
		Name: "KnownParam",
	}
	knownFlag.SetValue("KnownValue")
	knownFlag.SetAssigned(true)
	ctx.Flags().Add(knownFlag)

	// Initialize meta.Api with parameters
	api := meta.Api{
		Parameters: []meta.Parameter{
			{
				Name:     "KnownParam",
				Position: "Query",
			},
			{
				Name:     "UnknownParam",
				Position: "Query",
			},
		},
	}

	// Create Commando instance
	profile := config.Profile{
		Language: "en",
		RegionId: "cn-hangzhou",
	}
	commando := NewCommando(w, profile)

	// Call CheckApiParamWithBuildInArgs
	commando.CheckApiParamWithBuildInArgs(ctx, api)

	// Verify unknown flags
	unknownFlag, ok := ctx.UnknownFlags().GetValue("KnownParam")
	assert.True(t, ok)
	assert.Equal(t, "KnownValue", unknownFlag)
}

func TestDetectInConfigureMode(t *testing.T) {
	// Test case 1: No flags set
	flags := cli.NewFlagSet()
	result := DetectInConfigureMode(flags)
	assert.True(t, result, "Expected true when no flags are set")

	// Test case 2: Mode flag set
	flags = cli.NewFlagSet()
	modeFlag := &cli.Flag{Name: config.ModeFlagName}
	modeFlag.SetAssigned(true)
	flags.Add(modeFlag)
	result = DetectInConfigureMode(flags)
	assert.False(t, result, "Expected false when mode flag is set")

	// Test case 3: AccessKeyId flag set
	flags = cli.NewFlagSet()
	modeFlag.SetAssigned(true)
	flags.Add(modeFlag)
	akFlag := &cli.Flag{Name: config.AccessKeyIdFlagName}
	akFlag.SetAssigned(true)
	flags.Add(akFlag)
	result = DetectInConfigureMode(flags)
	assert.True(t, result, "Expected true when AccessKeyId flag is set")

	// Test case 4: AccessKeySecret flag set
	flags = cli.NewFlagSet()
	skFlag := &cli.Flag{Name: config.AccessKeySecretFlagName}
	skFlag.SetAssigned(true)
	flags.Add(skFlag)
	flags.Add(modeFlag)
	result = DetectInConfigureMode(flags)
	assert.True(t, result, "Expected true when AccessKeySecret flag is set")

	// Test case 5: StsToken flag set
	flags = cli.NewFlagSet()
	stsFlag := &cli.Flag{Name: config.StsTokenFlagName}
	stsFlag.SetAssigned(true)
	flags.Add(stsFlag)
	flags.Add(modeFlag)
	result = DetectInConfigureMode(flags)
	assert.True(t, result, "Expected true when StsToken flag is set")

	// Test case 6: RamRoleName flag set
	flags = cli.NewFlagSet()
	ramRoleNameFlag := &cli.Flag{Name: config.RamRoleNameFlagName}
	ramRoleNameFlag.SetAssigned(true)
	flags.Add(ramRoleNameFlag)
	flags.Add(modeFlag)
	result = DetectInConfigureMode(flags)
	assert.True(t, result, "Expected true when RamRoleName flag is set")

	// Test case 7: RamRoleArn flag set
	flags = cli.NewFlagSet()
	ramRoleArnFlag := &cli.Flag{Name: config.RamRoleArnFlagName}
	ramRoleArnFlag.SetAssigned(true)
	flags.Add(ramRoleArnFlag)
	flags.Add(modeFlag)
	result = DetectInConfigureMode(flags)
	assert.True(t, result, "Expected true when RamRoleArn flag is set")

	// Test case 8: RoleSessionName flag set
	flags = cli.NewFlagSet()
	roleSessionNameFlag := &cli.Flag{Name: config.RoleSessionNameFlagName}
	roleSessionNameFlag.SetAssigned(true)
	flags.Add(roleSessionNameFlag)
	flags.Add(modeFlag)
	result = DetectInConfigureMode(flags)
	assert.True(t, result, "Expected true when RoleSessionName flag is set")

	// Test case 9: PrivateKey flag set
	flags = cli.NewFlagSet()
	privateKeyFlag := &cli.Flag{Name: config.PrivateKeyFlagName}
	privateKeyFlag.SetAssigned(true)
	flags.Add(privateKeyFlag)
	flags.Add(modeFlag)
	result = DetectInConfigureMode(flags)
	assert.True(t, result, "Expected true when PrivateKey flag is set")

	// Test case 10: KeyPairName flag set
	flags = cli.NewFlagSet()
	keyPairNameFlag := &cli.Flag{Name: config.KeyPairNameFlagName}
	keyPairNameFlag.SetAssigned(true)
	flags.Add(keyPairNameFlag)
	flags.Add(modeFlag)
	result = DetectInConfigureMode(flags)
	assert.True(t, result, "Expected true when KeyPairName flag is set")

	// Test case 11: OIDCProviderARN flag set
	flags = cli.NewFlagSet()
	oidcProviderArnFlag := &cli.Flag{Name: config.OIDCProviderARNFlagName}
	oidcProviderArnFlag.SetAssigned(true)
	flags.Add(oidcProviderArnFlag)
	flags.Add(modeFlag)
	result = DetectInConfigureMode(flags)
	assert.True(t, result, "Expected true when OIDCProviderARN flag is set")

	// Test case 12: OIDCTokenFile flag set
	flags = cli.NewFlagSet()
	oidcTokenFileFlag := &cli.Flag{Name: config.OIDCTokenFileFlagName}
	oidcTokenFileFlag.SetAssigned(true)
	flags.Add(oidcTokenFileFlag)
	flags.Add(modeFlag)
	result = DetectInConfigureMode(flags)
	assert.True(t, result, "Expected true when OIDCTokenFile flag is set")
}

func TestProcessApiInvoke(t *testing.T) {
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)
	cmd := &cli.Command{}
	cmd.EnableUnknownFlag = true
	AddFlags(cmd.Flags())
	ctx.EnterCommand(cmd)

	profile := config.Profile{
		Language:        "en",
		Mode:            "AK",
		AccessKeyId:     "accesskeyid",
		AccessKeySecret: "accesskeysecret",
		RegionId:        "cn-hangzhou",
	}
	command := NewCommando(w, profile)

	t.Run("NilProduct", func(t *testing.T) {
		err := command.processApiInvoke(ctx, nil, nil, "GET", "/test")
		assert.Error(t, err)
		assert.Equal(t, "invalid product, please check product code", err.Error())
	})

	t.Run("CreateHttpContextError", func(t *testing.T) {
		product := &meta.Product{
			Code: "test",
		}
		err := command.processApiInvoke(ctx, product, nil, "INVALID", "/test")
		assert.Error(t, err)
	})

	t.Run("QuietFlagAssigned", func(t *testing.T) {
		product := &meta.Product{
			Code: "sls",
		}
		api := &meta.Api{
			Name: "TestApi",
			Product: &meta.Product{
				Version: "2017-08-01",
			},
		}
		QuietFlag(ctx.Flags()).SetAssigned(true)

		originCallHook := hookHttpContextCall
		originhook := hookHttpContextGetResponse
		defer func() {
			hookHttpContextGetResponse = originhook
			hookHttpContextCall = originCallHook
		}()
		hookHttpContextCall = func(fn func() error) func() error {
			return func() error {
				return nil
			}
		}
		hookHttpContextGetResponse = func(fn func() (string, error)) func() (string, error) {
			return func() (string, error) {
				return "test bytes", nil
			}
		}

		err := command.processApiInvoke(ctx, product, api, "GET", "/test")
		assert.NoError(t, err)
		assert.Empty(t, w.String())
	})

	t.Run("SuccessWithOutput", func(t *testing.T) {
		product := &meta.Product{
			Code: "sls",
		}
		api := &meta.Api{
			Name: "TestApi",
			Product: &meta.Product{
				Version: "2017-08-01",
			},
		}

		originCallHook := hookHttpContextCall
		originhook := hookHttpContextGetResponse
		defer func() {
			hookHttpContextGetResponse = originhook
			hookHttpContextCall = originCallHook
		}()
		hookHttpContextCall = func(fn func() error) func() error {
			return func() error {
				return nil
			}
		}
		hookHttpContextGetResponse = func(fn func() (string, error)) func() (string, error) {
			return func() (string, error) {
				return "test", nil
			}
		}
		err := command.processApiInvoke(ctx, product, api, "GET", "/test")
		assert.NoError(t, err)
	})

	t.Run("ProcessApiInvokeFail", func(t *testing.T) {
		product := &meta.Product{
			Code: "sls",
		}
		api := &meta.Api{
			Name: "PullLogs",
			Product: &meta.Product{
				Version: "2017-08-01",
			},
		}

		originCallHook := hookHttpContextCall
		defer func() {
			hookHttpContextCall = originCallHook
		}()
		hookHttpContextCall = func(fn func() error) func() error {
			return func() error {
				return errors.New("test error")
			}
		}
		err := command.processApiInvoke(ctx, product, api, "GET", "/PullLogs")
		assert.Equal(t, "test error", err.Error())
	})

	t.Run("ProcessApiInvokeNoOutput", func(t *testing.T) {
		product := &meta.Product{
			Code: "sls",
		}
		api := &meta.Api{
			Name: "PutLogs",
			Product: &meta.Product{
				Version: "2017-08-01",
			},
		}

		originCallHook := hookHttpContextCall
		originhook := hookHttpContextGetResponse
		defer func() {
			hookHttpContextGetResponse = originhook
			hookHttpContextCall = originCallHook
		}()
		hookHttpContextCall = func(fn func() error) func() error {
			return func() error {
				return nil
			}
		}
		hookHttpContextGetResponse = func(fn func() (string, error)) func() (string, error) {
			return func() (string, error) {
				return "", nil
			}
		}
		jsonData := `{
			"Logs": [
				{
					"Time": 1712345678,
					"Contents": [
						{ "Key": "method", "Value": "POST" },
						{ "Key": "path", "Value": "/api/login" }
					]
				}
			],
			"Topic": "web-logs",
			"Source": "192.168.1.100",
			"LogTags": [
				{ "Key": "env", "Value": "prod" }
			]
		}`
		BodyFlag(ctx.Flags()).SetAssigned(true)
		BodyFlag(ctx.Flags()).SetValue(string(jsonData))
		err := command.processApiInvoke(ctx, product, api, "GET", "/PutLogs")
		assert.NoError(t, err)
	})

	t.Run("ProcessApiInvokeGetResponseFail", func(t *testing.T) {
		product := &meta.Product{
			Code: "sls",
		}
		api := &meta.Api{
			Name: "PullLogs",
			Product: &meta.Product{
				Version: "2017-08-01",
			},
		}

		originCallHook := hookHttpContextCall
		originhook := hookHttpContextGetResponse
		defer func() {
			hookHttpContextGetResponse = originhook
			hookHttpContextCall = originCallHook
		}()
		hookHttpContextCall = func(fn func() error) func() error {
			return func() error {
				return nil
			}
		}
		hookHttpContextGetResponse = func(fn func() (string, error)) func() (string, error) {
			return func() (string, error) {
				return "", errors.New("test error")
			}
		}
		err := command.processApiInvoke(ctx, product, api, "GET", "/PullLogs")
		assert.Equal(t, "test error", err.Error())
	})

	t.Run("QueryFlagAssignedWithValidQuery", func(t *testing.T) {
		product := &meta.Product{
			Code: "sls",
		}
		api := &meta.Api{
			Name: "TestApi",
			Product: &meta.Product{
				Version: "2017-08-01",
			},
		}

		// Reset flags
		QuietFlag(ctx.Flags()).SetAssigned(false)
		QueryFlag(ctx.Flags()).SetAssigned(true)
		QueryFlag(ctx.Flags()).SetValue("key")

		originCallHook := hookHttpContextCall
		originhook := hookHttpContextGetResponse
		defer func() {
			hookHttpContextGetResponse = originhook
			hookHttpContextCall = originCallHook
			QueryFlag(ctx.Flags()).SetAssigned(false)
		}()
		hookHttpContextCall = func(fn func() error) func() error {
			return func() error {
				return nil
			}
		}
		hookHttpContextGetResponse = func(fn func() (string, error)) func() (string, error) {
			return func() (string, error) {
				return `{"key": "value"}`, nil
			}
		}

		w.Reset()
		err := command.processApiInvoke(ctx, product, api, "GET", "/test")
		assert.NoError(t, err)
		assert.Contains(t, w.String(), "value")
		assert.NotContains(t, w.String(), "key")
	})

	t.Run("QueryFlagAssignedWithQueryError", func(t *testing.T) {
		product := &meta.Product{
			Code: "sls",
		}
		api := &meta.Api{
			Name: "TestApi",
			Product: &meta.Product{
				Version: "2017-08-01",
			},
		}

		// Reset flags
		QuietFlag(ctx.Flags()).SetAssigned(false)
		QueryFlag(ctx.Flags()).SetAssigned(true)
		QueryFlag(ctx.Flags()).SetValue("invalid[")

		originCallHook := hookHttpContextCall
		originhook := hookHttpContextGetResponse
		defer func() {
			hookHttpContextGetResponse = originhook
			hookHttpContextCall = originCallHook
			QueryFlag(ctx.Flags()).SetAssigned(false)
		}()
		hookHttpContextCall = func(fn func() error) func() error {
			return func() error {
				return nil
			}
		}
		hookHttpContextGetResponse = func(fn func() (string, error)) func() (string, error) {
			return func() (string, error) {
				return `{"key": "value"}`, nil
			}
		}

		err := command.processApiInvoke(ctx, product, api, "GET", "/test")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "JMESPath query failed")
	})

	t.Run("QueryFlagNotAssigned", func(t *testing.T) {
		product := &meta.Product{
			Code: "sls",
		}
		api := &meta.Api{
			Name: "TestApi",
			Product: &meta.Product{
				Version: "2017-08-01",
			},
		}

		QuietFlag(ctx.Flags()).SetAssigned(false)

		originCallHook := hookHttpContextCall
		originhook := hookHttpContextGetResponse
		defer func() {
			hookHttpContextGetResponse = originhook
			hookHttpContextCall = originCallHook
		}()
		hookHttpContextCall = func(fn func() error) func() error {
			return func() error {
				return nil
			}
		}
		hookHttpContextGetResponse = func(fn func() (string, error)) func() (string, error) {
			return func() (string, error) {
				return `{"key": "value"}`, nil
			}
		}

		w.Reset()
		err := command.processApiInvoke(ctx, product, api, "GET", "/test")
		assert.NoError(t, err)
		assert.Contains(t, w.String(), "key")
		assert.Contains(t, w.String(), "value")
	})
}

func TestProcessApiInvokeFilterError(t *testing.T) {
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)
	cmd := &cli.Command{}
	cmd.EnableUnknownFlag = true
	AddFlags(cmd.Flags())
	ctx.EnterCommand(cmd)

	profile := config.Profile{
		Language:        "en",
		Mode:            "AK",
		AccessKeyId:     "accesskeyid",
		AccessKeySecret: "accesskeysecret",
		RegionId:        "cn-hangzhou",
	}
	command := NewCommando(w, profile)
	product := &meta.Product{
		Code: "sls",
	}
	api := &meta.Api{
		Name: "TestApi",
		Product: &meta.Product{
			Version: "2017-08-01",
		},
	}

	originCallHook := hookHttpContextCall
	originhook := hookHttpContextGetResponse
	defer func() {
		hookHttpContextGetResponse = originhook
		hookHttpContextCall = originCallHook
	}()
	hookHttpContextCall = func(fn func() error) func() error {
		return func() error {
			return nil
		}
	}
	hookHttpContextGetResponse = func(fn func() (string, error)) func() (string, error) {
		return func() (string, error) {
			return `{"Instances":{"Instance":[{"InstanceId":"i-123","InstanceName":"test-instance"}]}}`, nil
		}
	}
	OutputFlag(ctx.Flags()).SetAssigned(true)
	err := command.processApiInvoke(ctx, product, api, "GET", "/test")
	assert.Contains(t, err.Error(), "you need to assign col=col1,col2")
}

func TestCreateHttpContext(t *testing.T) {
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)
	cmd := &cli.Command{}
	cmd.EnableUnknownFlag = true
	AddFlags(cmd.Flags())
	ctx.EnterCommand(cmd)

	profile := config.Profile{
		Language:        "en",
		Mode:            "AK",
		AccessKeyId:     "accesskeyid",
		AccessKeySecret: "accesskeysecret",
		RegionId:        "cn-hangzhou",
	}
	command := NewCommando(w, profile)

	t.Run("NilProduct", func(t *testing.T) {
		invoker, err := command.createHttpContext(ctx, nil, nil, "GET", "/test")
		assert.Error(t, err)
		assert.Nil(t, invoker)
		assert.Equal(t, "invalid product, please check product code", err.Error())
	})

	t.Run("InvalidApiStyle", func(t *testing.T) {
		product := &meta.Product{
			Code:     "test",
			ApiStyle: "invalid",
		}
		invoker, err := command.createHttpContext(ctx, product, nil, "GET", "/test")
		assert.Error(t, err)
		assert.Nil(t, invoker)
		assert.Contains(t, err.Error(), "unchecked api style: invalid")
	})

	t.Run("ForceFlagWithUncheckedVersion", func(t *testing.T) {
		product := &meta.Product{
			Code:     "test",
			ApiStyle: "restful",
		}
		ForceFlag(ctx.Flags()).SetAssigned(true)
		VersionFlag(ctx.Flags()).SetAssigned(true)
		VersionFlag(ctx.Flags()).SetValue("2022-01-01")
		defer func() {
			ForceFlag(ctx.Flags()).SetAssigned(false)
			VersionFlag(ctx.Flags()).SetAssigned(false)
		}()

		invoker, err := command.createHttpContext(ctx, product, nil, "GET", "/test")
		assert.Error(t, err)
		assert.Nil(t, invoker)
		assert.Contains(t, err.Error(), "unchecked version 2022-01-01")
	})

	t.Run("ForceFlagWithStyleFlag", func(t *testing.T) {
		product := &meta.Product{
			Code:     "test",
			ApiStyle: "restful",
		}
		ForceFlag(ctx.Flags()).SetAssigned(true)
		VersionFlag(ctx.Flags()).SetAssigned(true)
		VersionFlag(ctx.Flags()).SetValue("2022-01-01")
		unknownFlag := &cli.Flag{
			Name: "style",
		}
		unknownFlag.SetValue("restful")
		unknownFlag.SetAssigned(true)
		ctx.Flags().Add(unknownFlag)
		defer func() {
			ForceFlag(ctx.Flags()).SetAssigned(false)
			VersionFlag(ctx.Flags()).SetAssigned(false)
		}()

		invoker, err := command.createHttpContext(ctx, product, nil, "test", "/test")
		assert.Error(t, err)
		assert.Nil(t, invoker)
		assert.Contains(t, err.Error(), "unchecked api style: restful or product: test")
	})

	t.Run("Success", func(t *testing.T) {
		product := &meta.Product{
			Code:     "sls",
			ApiStyle: "restful",
			Version:  "2017-08-01",
		}
		api := &meta.Api{
			Name: "TestApi",
		}

		invoker, _ := command.createHttpContext(ctx, product, api, "GET", "/test")
		assert.NotNil(t, invoker)
		openapiCtx, ok := invoker.(*OpenapiContext)
		assert.True(t, ok)
		assert.Equal(t, "GET", openapiCtx.method)
		assert.Equal(t, "/test", openapiCtx.path)
		assert.Equal(t, api, openapiCtx.api)
	})
}

func TestCreateHttpContextInitFail(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)
	cmd := &cli.Command{}
	cmd.EnableUnknownFlag = true
	AddFlags(cmd.Flags())
	ctx.EnterCommand(cmd)

	profile := config.Profile{
		Language:        "en",
		Mode:            "AK",
		AccessKeyId:     "accesskeyid",
		AccessKeySecret: "accesskeysecret",
	}
	command := NewCommando(stdout, profile)
	product := &meta.Product{
		Code:     "sls",
		ApiStyle: "restful",
		Version:  "2017-08-01",
	}
	api := &meta.Api{
		Name: "TestApi",
	}

	_, err := command.createHttpContext(ctx, product, api, "GET", "/test")
	assert.Contains(t, err.Error(), "init openapi client failed")
}

func TestCreateHttpContextRestCheckFail(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)
	cmd := &cli.Command{}
	cmd.EnableUnknownFlag = true
	AddFlags(cmd.Flags())
	ctx.EnterCommand(cmd)

	profile := config.Profile{
		Language:        "en",
		Mode:            "AK",
		AccessKeyId:     "accesskeyid",
		AccessKeySecret: "accesskeysecret",
		RegionId:        "cn-hangzhou",
	}
	command := NewCommando(stdout, profile)
	product := &meta.Product{
		Code:     "sls",
		ApiStyle: "restful",
		Version:  "2017-08-01",
	}
	api := &meta.Api{
		Name: "TestApi",
	}

	_, err := command.createHttpContext(ctx, product, api, "GET", "aaa/test")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "bad restful path aaa/test")
}

func TestMainForSlsProduct(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)
	profile := config.Profile{
		Language: "en",
		RegionId: "cn-hangzhou",
	}
	command := NewCommando(stdout, profile)
	assert.NotNil(t, command)

	cmd := &cli.Command{}
	AddFlags(cmd.Flags())
	cmd.EnableUnknownFlag = true
	command.InitWithCommand(cmd)

	t.Run("SLSProductWithOpenApi", func(t *testing.T) {
		originalFunc := meta.HookGetApi
		defer func() {
			meta.HookGetApi = originalFunc
		}()
		slsProduct := meta.Product{
			Code:     "sls",
			Version:  "2020-03-20",
			ApiStyle: "restful",
			ApiNames: []string{"TestApi"},
		}
		mockRepo, _ := meta.MockLoadRepository([]meta.Product{slsProduct})

		mockLibrary := &Library{
			builtinRepo: mockRepo,
		}
		command.library = mockLibrary

		meta.HookGetApi = func(fn func(productCode string, version string, apiName string) (meta.Api, bool)) func(productCode string, version string, apiName string) (meta.Api, bool) {
			return func(productCode string, version string, apiName string) (meta.Api, bool) {
				if productCode == "sls" && version == "2020-03-20" && apiName == "TestApi" {
					slsApi := meta.Api{
						Name:    "GetProject",
						Product: &meta.Product{Version: "2020-03-20"},
						Parameters: []meta.Parameter{
							{
								Name:     "TestHost",
								Position: "Host",
								Required: true,
							},
						},
					}
					return slsApi, true
				}
				return meta.Api{}, false
			}
		}

		stdout.Reset()
		ctx = cli.NewCommandContext(stdout, stderr)
		ctx.EnterCommand(cmd)

		// Add required flags for SLS
		regionflag := config.NewRegionFlag()
		regionflag.SetAssigned(true)
		regionflag.SetValue("cn-hangzhou")
		ctx.Flags().Add(regionflag)

		accessKeyIDFlag := config.NewAccessKeyIdFlag()
		accessKeyIDFlag.SetAssigned(true)
		accessKeyIDFlag.SetValue("test-access-key-id")
		ctx.Flags().Add(accessKeyIDFlag)

		accessKeySecretFlag := config.NewAccessKeySecretFlag()
		accessKeySecretFlag.SetAssigned(true)
		accessKeySecretFlag.SetValue("test-access-key-secret")
		ctx.Flags().Add(accessKeySecretFlag)

		// Test the SLS product call that should use OpenAPI
		args := []string{"sls", "TestApi"}
		err := command.main(ctx, args)
		assert.Equal(t, err.Error(), "product 'sls' need proper restful call with ApiName or {GET|PUT|POST|DELETE} <path>")
	})

	t.Run("SLSProductInvalidRestCall", func(t *testing.T) {
		originalFunc := meta.HookGetApiByPath
		defer func() {
			meta.HookGetApiByPath = originalFunc
		}()
		slsProduct := meta.Product{
			Code:     "sls",
			Version:  "2020-03-20",
			ApiStyle: "restful",
			ApiNames: []string{"TestApi"},
		}
		mockRepo, _ := meta.MockLoadRepository([]meta.Product{slsProduct})

		mockLibrary := &Library{
			builtinRepo: mockRepo,
		}
		command.library = mockLibrary

		meta.HookGetApiByPath = func(fn func(productCode string, version string, method string, path string) (meta.Api, bool)) func(productCode string, version string, method string, path string) (meta.Api, bool) {
			return func(productCode string, version string, method string, path string) (meta.Api, bool) {
				if productCode == "sls" && version == "2020-03-20" && method == "Get" && path == "/" {
					slsApi := meta.Api{
						Name:    "GetProject",
						Product: &meta.Product{Version: "2020-03-20"},
						Parameters: []meta.Parameter{
							{
								Name:     "TestParam",
								Position: "Query",
								Required: false,
							},
						},
					}
					return slsApi, true
				}
				return meta.Api{}, false
			}
		}

		// Set up context for SLS product call
		stdout.Reset()
		ctx = cli.NewCommandContext(stdout, stderr)
		ctx.EnterCommand(cmd)

		// Add required flags for SLS
		regionflag := config.NewRegionFlag()
		regionflag.SetAssigned(true)
		regionflag.SetValue("cn-hangzhou")
		ctx.Flags().Add(regionflag)

		accessKeyIDFlag := config.NewAccessKeyIdFlag()
		accessKeyIDFlag.SetAssigned(true)
		accessKeyIDFlag.SetValue("test-access-key-id")
		ctx.Flags().Add(accessKeyIDFlag)

		accessKeySecretFlag := config.NewAccessKeySecretFlag()
		accessKeySecretFlag.SetAssigned(true)
		accessKeySecretFlag.SetValue("test-access-key-secret")
		ctx.Flags().Add(accessKeySecretFlag)

		args := []string{"sls", "Get", "/"}
		err := command.main(ctx, args)
		assert.Contains(t, err.Error(), "too broad path: / for method: Get, please use specific ApiName instead")
	})

	t.Run("SLSProductWithRestCall", func(t *testing.T) {
		originalFunc := meta.HookGetApiByPath
		defer func() {
			meta.HookGetApiByPath = originalFunc
		}()
		slsProduct := meta.Product{
			Code:     "sls",
			Version:  "2020-03-20",
			ApiStyle: "restful",
			ApiNames: []string{"TestApi"},
		}
		mockRepo, _ := meta.MockLoadRepository([]meta.Product{slsProduct})

		mockLibrary := &Library{
			builtinRepo: mockRepo,
		}
		command.library = mockLibrary

		meta.HookGetApiByPath = func(fn func(productCode string, version string, method string, path string) (meta.Api, bool)) func(productCode string, version string, method string, path string) (meta.Api, bool) {
			return func(productCode string, version string, method string, path string) (meta.Api, bool) {
				if productCode == "sls" && version == "2020-03-20" && method == "Gets" && path == "/abc" {
					slsApi := meta.Api{
						Name: "GetProject",
						Parameters: []meta.Parameter{
							{
								Name:     "TestParam",
								Position: "Query",
								Required: false,
							},
						},
					}
					return slsApi, true
				}
				return meta.Api{}, false
			}
		}

		// Set up context for SLS product call
		stdout.Reset()
		ctx = cli.NewCommandContext(stdout, stderr)
		ctx.EnterCommand(cmd)

		// Add required flags for SLS
		regionflag := config.NewRegionFlag()
		regionflag.SetAssigned(true)
		regionflag.SetValue("cn-hangzhou")
		ctx.Flags().Add(regionflag)

		accessKeyIDFlag := config.NewAccessKeyIdFlag()
		accessKeyIDFlag.SetAssigned(true)
		accessKeyIDFlag.SetValue("test-access-key-id")
		ctx.Flags().Add(accessKeyIDFlag)

		accessKeySecretFlag := config.NewAccessKeySecretFlag()
		accessKeySecretFlag.SetAssigned(true)
		accessKeySecretFlag.SetValue("test-access-key-secret")
		ctx.Flags().Add(accessKeySecretFlag)

		args := []string{"sls", "Gets", "/abc"}
		err := command.main(ctx, args)
		assert.Contains(t, err.Error(), "product 'sls' need proper restful call with ApiName or {GET|PUT|POST|DELETE} <path>")
	})
}

func TestMainForNonSlsProductApi(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	profile := config.Profile{
		Language: "en",
		RegionId: "cn-hangzhou",
	}
	command := NewCommando(stdout, profile)
	assert.NotNil(t, command)

	cmd := &cli.Command{}
	AddFlags(cmd.Flags())
	cmd.EnableUnknownFlag = true
	command.InitWithCommand(cmd)
	originalFunc := meta.HookGetApi
	defer func() {
		meta.HookGetApi = originalFunc
	}()
	ecsProduct := meta.Product{
		Code:     "ecs",
		Version:  "2020-03-20",
		ApiStyle: "restful",
		ApiNames: []string{"TestApi"},
	}
	mockRepo, _ := meta.MockLoadRepository([]meta.Product{ecsProduct})

	mockLibrary := &Library{
		builtinRepo: mockRepo,
	}
	command.library = mockLibrary

	meta.HookGetApi = func(fn func(productCode string, version string, apiName string) (meta.Api, bool)) func(productCode string, version string, apiName string) (meta.Api, bool) {
		return func(productCode string, version string, apiName string) (meta.Api, bool) {
			if productCode == "ecs" {
				ecsApi := meta.Api{
					Name:    "GetProject",
					Product: &meta.Product{Version: "2020-03-20"},
					Parameters: []meta.Parameter{
						{
							Name:     "TestHost",
							Position: "Host",
							Required: true,
						},
					},
				}
				return ecsApi, true
			}
			return meta.Api{}, false
		}
	}

	stdout.Reset()
	ctx := cli.NewCommandContext(stdout, stderr)
	ctx.EnterCommand(cmd)

	regionflag := config.NewRegionFlag()
	regionflag.SetAssigned(true)
	regionflag.SetValue("cn-hangzhou")
	ctx.Flags().Add(regionflag)

	accessKeyIDFlag := config.NewAccessKeyIdFlag()
	accessKeyIDFlag.SetAssigned(true)
	accessKeyIDFlag.SetValue("test-access-key-id")
	ctx.Flags().Add(accessKeyIDFlag)

	accessKeySecretFlag := config.NewAccessKeySecretFlag()
	accessKeySecretFlag.SetAssigned(true)
	accessKeySecretFlag.SetValue("test-access-key-secret")
	ctx.Flags().Add(accessKeySecretFlag)

	args := []string{"ecs", "TestApi"}
	err := command.main(ctx, args)
	assert.NotNil(t, err)
}

// Regression test: for a restful product, when the user provides an API name that does not exist in metadata (e.g. `aliyun apig GetPlugin`), the error should be `InvalidApiError` with suggestions,
// NOT the confusing `product 'xxx' need restful call` produced by checkRestfulMethod.
func TestMainRestfulProductWithInvalidApiName(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	profile := config.Profile{
		Language: "en",
		RegionId: "cn-hangzhou",
	}
	command := NewCommando(stdout, profile)
	assert.NotNil(t, command)

	cmd := &cli.Command{}
	AddFlags(cmd.Flags())
	cmd.EnableUnknownFlag = true
	command.InitWithCommand(cmd)

	originalFunc := meta.HookGetApi
	defer func() {
		meta.HookGetApi = originalFunc
	}()

	apigProduct := meta.Product{
		Code:     "APIG",
		Version:  "2024-03-27",
		ApiStyle: "restful",
		ApiNames: []string{"ListPlugins", "GetPluginAttachment", "InstallPlugin"},
	}
	mockRepo, _ := meta.MockLoadRepository([]meta.Product{apigProduct})
	command.library = &Library{builtinRepo: mockRepo}

	meta.HookGetApi = func(fn func(productCode string, version string, apiName string) (meta.Api, bool)) func(productCode string, version string, apiName string) (meta.Api, bool) {
		return func(productCode string, version string, apiName string) (meta.Api, bool) {
			return meta.Api{}, false
		}
	}

	ctx := cli.NewCommandContext(stdout, stderr)
	ctx.EnterCommand(cmd)

	regionflag := config.NewRegionFlag()
	regionflag.SetAssigned(true)
	regionflag.SetValue("cn-hangzhou")
	ctx.Flags().Add(regionflag)

	accessKeyIDFlag := config.NewAccessKeyIdFlag()
	accessKeyIDFlag.SetAssigned(true)
	accessKeyIDFlag.SetValue("test-access-key-id")
	ctx.Flags().Add(accessKeyIDFlag)

	accessKeySecretFlag := config.NewAccessKeySecretFlag()
	accessKeySecretFlag.SetAssigned(true)
	accessKeySecretFlag.SetValue("test-access-key-secret")
	ctx.Flags().Add(accessKeySecretFlag)

	args := []string{"apig", "GetPlugin"}
	err := command.main(ctx, args)
	assert.NotNil(t, err)
	// Must be InvalidApiError, not the generic "need restful call" message.
	invalidApiErr, ok := err.(*InvalidApiError)
	assert.True(t, ok, "expected *InvalidApiError, got %T: %v", err, err)
	assert.Equal(t, "GetPlugin", invalidApiErr.Name)
	assert.Contains(t, err.Error(), "'GetPlugin' is not a valid api")
	assert.NotContains(t, err.Error(), "need restful call")
}

func TestMainForNonSlsProductApiWithRestCall(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	profile := config.Profile{
		Language: "en",
		RegionId: "cn-hangzhou",
	}
	command := NewCommando(stdout, profile)

	cmd := &cli.Command{}
	AddFlags(cmd.Flags())
	cmd.EnableUnknownFlag = true
	command.InitWithCommand(cmd)

	originalFunc := meta.HookGetApiByPath
	defer func() {
		meta.HookGetApiByPath = originalFunc
	}()
	ecsProduct := meta.Product{
		Code:     "ecs",
		Version:  "2020-03-20",
		ApiStyle: "restful",
		ApiNames: []string{"TestApi"},
	}
	mockRepo, _ := meta.MockLoadRepository([]meta.Product{ecsProduct})

	mockLibrary := &Library{
		builtinRepo: mockRepo,
	}
	command.library = mockLibrary

	meta.HookGetApiByPath = func(fn func(productCode string, version string, method string, path string) (meta.Api, bool)) func(productCode string, version string, method string, path string) (meta.Api, bool) {
		return func(productCode string, version string, method string, path string) (meta.Api, bool) {
			if productCode == "ecs" {
				ecsApi := meta.Api{
					Name: "GetProject",
					Parameters: []meta.Parameter{
						{
							Name:     "TestParam",
							Position: "Query",
							Required: false,
						},
					},
				}
				return ecsApi, true
			}
			return meta.Api{}, false
		}
	}

	// Set up context for SLS product call
	stdout.Reset()
	ctx := cli.NewCommandContext(stdout, stderr)
	ctx.EnterCommand(cmd)

	// Add required flags for SLS
	regionflag := config.NewRegionFlag()
	regionflag.SetAssigned(true)
	regionflag.SetValue("cn-hangzhou")
	ctx.Flags().Add(regionflag)

	accessKeyIDFlag := config.NewAccessKeyIdFlag()
	accessKeyIDFlag.SetAssigned(true)
	accessKeyIDFlag.SetValue("test-access-key-id")
	ctx.Flags().Add(accessKeyIDFlag)

	accessKeySecretFlag := config.NewAccessKeySecretFlag()
	accessKeySecretFlag.SetAssigned(true)
	accessKeySecretFlag.SetValue("test-access-key-secret")
	ctx.Flags().Add(accessKeySecretFlag)

	args := []string{"ecs", "Gets", "/abc"}
	err := command.main(ctx, args)
	assert.NotNil(t, err)
}

func TestApplyQueryFilter(t *testing.T) {
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)
	AddFlags(ctx.Flags())

	t.Run("QueryFlagNotAssigned", func(t *testing.T) {
		output := `{"key": "value"}`
		result, err := ApplyQueryFilter(ctx, output)
		assert.NoError(t, err)
		assert.Equal(t, output, result)
	})

	t.Run("QueryFlagEmpty", func(t *testing.T) {
		queryFlag := QueryFlag(ctx.Flags())
		queryFlag.SetAssigned(true)
		queryFlag.SetValue("")
		output := `{"key": "value"}`
		result, err := ApplyQueryFilter(ctx, output)
		assert.NoError(t, err)
		assert.Equal(t, output, result)
	})

	t.Run("EmptyOutput", func(t *testing.T) {
		queryFlag := QueryFlag(ctx.Flags())
		queryFlag.SetAssigned(true)
		queryFlag.SetValue("key")
		output := ""
		result, err := ApplyQueryFilter(ctx, output)
		assert.NoError(t, err)
		assert.Equal(t, output, result)
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		queryFlag := QueryFlag(ctx.Flags())
		queryFlag.SetAssigned(true)
		queryFlag.SetValue("key")
		output := `invalid json`
		result, err := ApplyQueryFilter(ctx, output)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse JSON response")
		assert.Equal(t, output, result)
	})

	t.Run("ValidQuerySimpleField", func(t *testing.T) {
		queryFlag := QueryFlag(ctx.Flags())
		queryFlag.SetAssigned(true)
		queryFlag.SetValue("key")
		output := `{"key": "value"}`
		result, err := ApplyQueryFilter(ctx, output)
		assert.NoError(t, err)
		assert.Equal(t, `"value"`, result)
	})

	t.Run("ValidQueryNestedField", func(t *testing.T) {
		queryFlag := QueryFlag(ctx.Flags())
		queryFlag.SetAssigned(true)
		queryFlag.SetValue("nested.field")
		output := `{"nested": {"field": "value"}}`
		result, err := ApplyQueryFilter(ctx, output)
		assert.NoError(t, err)
		assert.Equal(t, `"value"`, result)
	})

	t.Run("ValidQueryArray", func(t *testing.T) {
		queryFlag := QueryFlag(ctx.Flags())
		queryFlag.SetAssigned(true)
		queryFlag.SetValue("items[*].name")
		output := `{"items": [{"name": "item1"}, {"name": "item2"}]}`
		result, err := ApplyQueryFilter(ctx, output)
		assert.NoError(t, err)
		// Result should be JSON array
		assert.Contains(t, result, "item1")
		assert.Contains(t, result, "item2")
	})

	t.Run("ValidQueryArrayOfArrays", func(t *testing.T) {
		queryFlag := QueryFlag(ctx.Flags())
		queryFlag.SetAssigned(true)
		queryFlag.SetValue("instances[*].[instanceId,status]")
		output := `{"instances": [{"instanceId": "i-xxx", "status": "Running"}, {"instanceId": "i-yyy", "status": "Stopped"}]}`
		result, err := ApplyQueryFilter(ctx, output)
		assert.NoError(t, err)
		// Result should be array of arrays
		assert.Contains(t, result, "i-xxx")
		assert.Contains(t, result, "Running")
		assert.Contains(t, result, "i-yyy")
		assert.Contains(t, result, "Stopped")
	})

	t.Run("ValidQueryObjectArray", func(t *testing.T) {
		queryFlag := QueryFlag(ctx.Flags())
		queryFlag.SetAssigned(true)
		queryFlag.SetValue("instances[*].{id: instanceId, name: instanceName}")
		output := `{"instances": [{"instanceId": "i-xxx", "instanceName": "test"}, {"instanceId": "i-yyy", "instanceName": "prod"}]}`
		result, err := ApplyQueryFilter(ctx, output)
		assert.NoError(t, err)
		// Result should be array of objects
		assert.Contains(t, result, "i-xxx")
		assert.Contains(t, result, "test")
		assert.Contains(t, result, "i-yyy")
		assert.Contains(t, result, "prod")
	})

	t.Run("InvalidJMESPath", func(t *testing.T) {
		queryFlag := QueryFlag(ctx.Flags())
		queryFlag.SetAssigned(true)
		queryFlag.SetValue("invalid[")
		output := `{"key": "value"}`
		result, err := ApplyQueryFilter(ctx, output)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "JMESPath query failed")
		assert.Equal(t, output, result)
	})

	t.Run("QueryReturnsNull", func(t *testing.T) {
		queryFlag := QueryFlag(ctx.Flags())
		queryFlag.SetAssigned(true)
		queryFlag.SetValue("nonexistent")
		output := `{"key": "value"}`
		result, err := ApplyQueryFilter(ctx, output)
		assert.NoError(t, err)
		assert.Equal(t, "null", result)
	})

	t.Run("QueryReturnsNumber", func(t *testing.T) {
		queryFlag := QueryFlag(ctx.Flags())
		queryFlag.SetAssigned(true)
		queryFlag.SetValue("count")
		output := `{"count": 42}`
		result, err := ApplyQueryFilter(ctx, output)
		assert.NoError(t, err)
		assert.Equal(t, "42", result)
	})

	t.Run("QueryReturnsBoolean", func(t *testing.T) {
		queryFlag := QueryFlag(ctx.Flags())
		queryFlag.SetAssigned(true)
		queryFlag.SetValue("enabled")
		output := `{"enabled": true}`
		result, err := ApplyQueryFilter(ctx, output)
		assert.NoError(t, err)
		assert.Equal(t, "true", result)
	})

	t.Run("QueryReturnsArray", func(t *testing.T) {
		queryFlag := QueryFlag(ctx.Flags())
		queryFlag.SetAssigned(true)
		queryFlag.SetValue("items")
		output := `{"items": [1, 2, 3]}`
		result, err := ApplyQueryFilter(ctx, output)
		assert.NoError(t, err)
		assert.Equal(t, "[1,2,3]", result)
	})

	t.Run("QueryReturnsObject", func(t *testing.T) {
		queryFlag := QueryFlag(ctx.Flags())
		queryFlag.SetAssigned(true)
		queryFlag.SetValue("metadata")
		output := `{"metadata": {"key": "value"}}`
		result, err := ApplyQueryFilter(ctx, output)
		assert.NoError(t, err)
		assert.Contains(t, result, "key")
		assert.Contains(t, result, "value")
	})

	t.Run("ComplexNestedQuery", func(t *testing.T) {
		queryFlag := QueryFlag(ctx.Flags())
		queryFlag.SetAssigned(true)
		queryFlag.SetValue("data.items[0].name")
		output := `{"data": {"items": [{"name": "first"}, {"name": "second"}]}}`
		result, err := ApplyQueryFilter(ctx, output)
		assert.NoError(t, err)
		assert.Equal(t, `"first"`, result)
	})

	t.Run("QueryWithFilter", func(t *testing.T) {
		queryFlag := QueryFlag(ctx.Flags())
		queryFlag.SetAssigned(true)
		queryFlag.SetValue("items[?status=='active'].name")
		output := `{"items": [{"name": "item1", "status": "active"}, {"name": "item2", "status": "inactive"}]}`
		result, err := ApplyQueryFilter(ctx, output)
		assert.NoError(t, err)
		assert.Contains(t, result, "item1")
		assert.NotContains(t, result, "item2")
	})

	t.Run("QueryToEntriesWithFilter", func(t *testing.T) {
		queryFlag := QueryFlag(ctx.Flags())
		queryFlag.SetAssigned(true)
		queryFlag.SetValue("to_entries(items)")
		output := `{"items": {"name": "item1", "status": "active"}}`
		result, err := ApplyQueryFilter(ctx, output)
		assert.NoError(t, err)
		assert.Equal(t, `[{"key":"name","value":"item1"},{"key":"status","value":"active"}]`, result)
	})
}

func TestMain_RestfulCallWithForceAndApiFinding(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	profile := config.Profile{
		Language:        "en",
		Mode:            "AK",
		AccessKeyId:     "test-access-key-id",
		AccessKeySecret: "test-access-key-secret",
		RegionId:        "cn-hangzhou",
	}
	command := NewCommando(stdout, profile)

	cmd := &cli.Command{}
	AddFlags(cmd.Flags())
	cmd.EnableUnknownFlag = true
	command.InitWithCommand(cmd)

	// Mock product
	ecsProduct := meta.Product{
		Code:     "ecs",
		Version:  "2014-05-26",
		ApiStyle: "restful",
		ApiNames: []string{"DescribeInstances"},
	}
	mockRepo, _ := meta.MockLoadRepository([]meta.Product{ecsProduct})
	mockLibrary := &Library{
		builtinRepo: mockRepo,
	}
	command.library = mockLibrary

	// Set environment variable to skip profile file loading
	// This allows LoadProfileWithContext to use flags directly instead of loading from file
	originalIgnoreProfile := os.Getenv("ALIBABA_CLOUD_IGNORE_PROFILE")
	os.Setenv("ALIBABA_CLOUD_IGNORE_PROFILE", "TRUE")
	defer func() {
		if originalIgnoreProfile == "" {
			os.Unsetenv("ALIBABA_CLOUD_IGNORE_PROFILE")
		} else {
			os.Setenv("ALIBABA_CLOUD_IGNORE_PROFILE", originalIgnoreProfile)
		}
	}()

	originalHook := meta.HookGetApiByPath
	defer func() {
		meta.HookGetApiByPath = originalHook
	}()

	t.Run("ApiNotFoundWithoutForce", func(t *testing.T) {
		meta.HookGetApiByPath = func(fn func(productCode string, version string, method string, path string) (meta.Api, bool)) func(productCode string, version string, method string, path string) (meta.Api, bool) {
			return func(productCode string, version string, method string, path string) (meta.Api, bool) {
				return meta.Api{}, false // API not found
			}
		}

		ctx := cli.NewCommandContext(stdout, stderr)
		ctx.EnterCommand(cmd)
		regionflag := config.NewRegionFlag()
		regionflag.SetAssigned(true)
		regionflag.SetValue("cn-hangzhou")
		ctx.Flags().Add(regionflag)
		accessKeyIDFlag := config.NewAccessKeyIdFlag()
		accessKeyIDFlag.SetAssigned(true)
		accessKeyIDFlag.SetValue("test-access-key-id")
		ctx.Flags().Add(accessKeyIDFlag)
		accessKeySecretFlag := config.NewAccessKeySecretFlag()
		accessKeySecretFlag.SetAssigned(true)
		accessKeySecretFlag.SetValue("test-access-key-secret")
		ctx.Flags().Add(accessKeySecretFlag)
		ForceFlag(ctx.Flags()).SetAssigned(false) // No force flag

		args := []string{"ecs", "GET", "/nonexistent"}
		err := command.main(ctx, args)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "can not find api by path")
		assert.Contains(t, err.Error(), "/nonexistent")
	})

	t.Run("ApiNotFoundWithForce", func(t *testing.T) {
		meta.HookGetApiByPath = func(fn func(productCode string, version string, method string, path string) (meta.Api, bool)) func(productCode string, version string, method string, path string) (meta.Api, bool) {
			return func(productCode string, version string, method string, path string) (meta.Api, bool) {
				return meta.Api{}, false // API not found
			}
		}

		ctx := cli.NewCommandContext(stdout, stderr)
		ctx.EnterCommand(cmd)
		regionflag := config.NewRegionFlag()
		regionflag.SetAssigned(true)
		regionflag.SetValue("cn-hangzhou")
		ctx.Flags().Add(regionflag)
		accessKeyIDFlag := config.NewAccessKeyIdFlag()
		accessKeyIDFlag.SetAssigned(true)
		accessKeyIDFlag.SetValue("test-access-key-id")
		ctx.Flags().Add(accessKeyIDFlag)
		accessKeySecretFlag := config.NewAccessKeySecretFlag()
		accessKeySecretFlag.SetAssigned(true)
		accessKeySecretFlag.SetValue("test-access-key-secret")
		ctx.Flags().Add(accessKeySecretFlag)
		skipflag := config.NewSkipSecureVerify()
		skipflag.SetAssigned(true)
		ctx.Flags().Add(skipflag)
		endpointflag := config.NewEndpointFlag()
		endpointflag.SetAssigned(true)
		endpointflag.SetValue("ecs.cn-hangzhou.aliyuncs.com")
		ctx.Flags().Add(endpointflag)
		ForceFlag(ctx.Flags()).SetAssigned(true) // Force flag enabled

		originalHookdo := hookdo
		defer func() {
			hookdo = originalHookdo
		}()
		hookdo = func(fn func() (*responses.CommonResponse, error)) func() (*responses.CommonResponse, error) {
			return func() (*responses.CommonResponse, error) {
				resp := responses.NewCommonResponse()
				return resp, nil
			}
		}

		DryRunFlag(ctx.Flags()).SetAssigned(true)
		args := []string{"ecs", "GET", "/nonexistent"}
		err := command.main(ctx, args)
		assert.NoError(t, err) // succeed with force flag
	})

	t.Run("ApiFoundCallsCheckApiParamWithBuildInArgs", func(t *testing.T) {
		meta.HookGetApiByPath = func(fn func(productCode string, version string, method string, path string) (meta.Api, bool)) func(productCode string, version string, method string, path string) (meta.Api, bool) {
			return func(productCode string, version string, method string, path string) (meta.Api, bool) {
				if productCode == "ecs" && method == "GET" && path == "/instances" {
					return meta.Api{
						Name: "DescribeInstances",
						Product: &meta.Product{
							Code:    "ecs",
							Version: "2014-05-26",
						},
						Parameters: []meta.Parameter{
							{
								Name:     "RegionId",
								Position: "Query",
								Required: true,
							},
						},
					}, true
				}
				return meta.Api{}, false
			}
		}

		ctx := cli.NewCommandContext(stdout, stderr)
		ctx.EnterCommand(cmd)
		regionflag := config.NewRegionFlag()
		regionflag.SetAssigned(true)
		regionflag.SetValue("cn-hangzhou")
		ctx.Flags().Add(regionflag)
		accessKeyIDFlag := config.NewAccessKeyIdFlag()
		accessKeyIDFlag.SetAssigned(true)
		accessKeyIDFlag.SetValue("test-access-key-id")
		ctx.Flags().Add(accessKeyIDFlag)
		accessKeySecretFlag := config.NewAccessKeySecretFlag()
		accessKeySecretFlag.SetAssigned(true)
		accessKeySecretFlag.SetValue("test-access-key-secret")
		ctx.Flags().Add(accessKeySecretFlag)
		skipflag := config.NewSkipSecureVerify()
		skipflag.SetAssigned(true)
		ctx.Flags().Add(skipflag)
		endpointflag := config.NewEndpointFlag()
		endpointflag.SetAssigned(true)
		endpointflag.SetValue("ecs.cn-hangzhou.aliyuncs.com")
		ctx.Flags().Add(endpointflag)

		// Add RegionId as a known flag to test CheckApiParamWithBuildInArgs
		regionIdFlag := &cli.Flag{Name: "RegionId"}
		regionIdFlag.SetValue("cn-hangzhou")
		regionIdFlag.SetAssigned(true)
		ctx.Flags().Add(regionIdFlag)

		originalHookdo := hookdo
		defer func() {
			hookdo = originalHookdo
		}()
		hookdo = func(fn func() (*responses.CommonResponse, error)) func() (*responses.CommonResponse, error) {
			return func() (*responses.CommonResponse, error) {
				resp := responses.NewCommonResponse()
				return resp, nil
			}
		}

		DryRunFlag(ctx.Flags()).SetAssigned(true)
		args := []string{"ecs", "GET", "/instances"}
		err := command.main(ctx, args)
		assert.NoError(t, err)

		unknownFlag, ok := ctx.UnknownFlags().GetValue("RegionId")
		assert.True(t, ok, "RegionId should be copied to UnknownFlags by CheckApiParamWithBuildInArgs")
		assert.Equal(t, "cn-hangzhou", unknownFlag)
	})

	t.Run("ApiFoundWithShouldUseOpenapi", func(t *testing.T) {
		slsProduct := meta.Product{
			Code:     "sls",
			Version:  "2020-03-20",
			ApiStyle: "restful",
			ApiNames: []string{"GetProject"},
		}
		mockRepo, _ := meta.MockLoadRepository([]meta.Product{slsProduct})
		mockLibrary := &Library{
			builtinRepo: mockRepo,
		}
		command.library = mockLibrary

		processApiInvokeCalled := false
		meta.HookGetApiByPath = func(fn func(productCode string, version string, method string, path string) (meta.Api, bool)) func(productCode string, version string, method string, path string) (meta.Api, bool) {
			return func(productCode string, version string, method string, path string) (meta.Api, bool) {
				if productCode == "sls" && method == "GET" && path == "/projects" {
					return meta.Api{
						Name: "GetProject",
						Product: &meta.Product{
							Code:    "sls",
							Version: "2020-03-20",
						},
					}, true
				}
				return meta.Api{}, false
			}
		}

		originalHookCall := hookHttpContextCall
		originalHookGetResponse := hookHttpContextGetResponse
		defer func() {
			hookHttpContextCall = originalHookCall
			hookHttpContextGetResponse = originalHookGetResponse
		}()
		hookHttpContextCall = func(fn func() error) func() error {
			processApiInvokeCalled = true
			return func() error {
				return nil
			}
		}
		hookHttpContextGetResponse = func(fn func() (string, error)) func() (string, error) {
			return func() (string, error) {
				return "{}", nil
			}
		}

		ctx := cli.NewCommandContext(stdout, stderr)
		ctx.EnterCommand(cmd)
		regionflag := config.NewRegionFlag()
		regionflag.SetAssigned(true)
		regionflag.SetValue("cn-hangzhou")
		ctx.Flags().Add(regionflag)
		accessKeyIDFlag := config.NewAccessKeyIdFlag()
		accessKeyIDFlag.SetAssigned(true)
		accessKeyIDFlag.SetValue("test-access-key-id")
		ctx.Flags().Add(accessKeyIDFlag)
		accessKeySecretFlag := config.NewAccessKeySecretFlag()
		accessKeySecretFlag.SetAssigned(true)
		accessKeySecretFlag.SetValue("test-access-key-secret")
		ctx.Flags().Add(accessKeySecretFlag)

		args := []string{"sls", "GET", "/projects"}
		err := command.main(ctx, args)
		assert.NoError(t, err)
		assert.True(t, processApiInvokeCalled, "processApiInvoke should be called for SLS product")
	})

	t.Run("ApiFoundWithoutShouldUseOpenapi", func(t *testing.T) {
		ecsProduct := meta.Product{
			Code:     "ecs",
			Version:  "2014-05-26",
			ApiStyle: "restful",
			ApiNames: []string{"DescribeInstances"},
		}
		mockRepo, _ := meta.MockLoadRepository([]meta.Product{ecsProduct})
		mockLibrary := &Library{
			builtinRepo: mockRepo,
		}
		command.library = mockLibrary

		meta.HookGetApiByPath = func(fn func(productCode string, version string, method string, path string) (meta.Api, bool)) func(productCode string, version string, method string, path string) (meta.Api, bool) {
			return func(productCode string, version string, method string, path string) (meta.Api, bool) {
				if productCode == "ecs" && method == "GET" && path == "/instances" {
					return meta.Api{
						Name: "DescribeInstances",
						Product: &meta.Product{
							Code:    "ecs",
							Version: "2014-05-26",
						},
					}, true
				}
				return meta.Api{}, false
			}
		}

		originalHookdo := hookdo
		defer func() {
			hookdo = originalHookdo
		}()
		hookdo = func(fn func() (*responses.CommonResponse, error)) func() (*responses.CommonResponse, error) {
			return func() (*responses.CommonResponse, error) {
				resp := responses.NewCommonResponse()
				return resp, nil
			}
		}

		ctx := cli.NewCommandContext(stdout, stderr)
		ctx.EnterCommand(cmd)
		regionflag := config.NewRegionFlag()
		regionflag.SetAssigned(true)
		regionflag.SetValue("cn-hangzhou")
		ctx.Flags().Add(regionflag)
		accessKeyIDFlag := config.NewAccessKeyIdFlag()
		accessKeyIDFlag.SetAssigned(true)
		accessKeyIDFlag.SetValue("test-access-key-id")
		ctx.Flags().Add(accessKeyIDFlag)
		accessKeySecretFlag := config.NewAccessKeySecretFlag()
		accessKeySecretFlag.SetAssigned(true)
		accessKeySecretFlag.SetValue("test-access-key-secret")
		ctx.Flags().Add(accessKeySecretFlag)
		skipflag := config.NewSkipSecureVerify()
		skipflag.SetAssigned(true)
		ctx.Flags().Add(skipflag)
		endpointflag := config.NewEndpointFlag()
		endpointflag.SetAssigned(true)
		endpointflag.SetValue("ecs.cn-hangzhou.aliyuncs.com")
		ctx.Flags().Add(endpointflag)

		DryRunFlag(ctx.Flags()).SetAssigned(true)
		args := []string{"ecs", "GET", "/instances"}
		err := command.main(ctx, args)
		assert.NoError(t, err)
	})

	t.Run("ApiNotFoundWithShouldUseOpenapi", func(t *testing.T) {
		// Test the specific error path at lines 188-190
		// This happens when ShouldUseOpenapi is true (SLS product) and API is not found
		slsProduct := meta.Product{
			Code:     "sls",
			Version:  "2020-03-20",
			ApiStyle: "restful",
			ApiNames: []string{"GetProject"},
		}
		mockRepo, _ := meta.MockLoadRepository([]meta.Product{slsProduct})
		mockLibrary := &Library{
			builtinRepo: mockRepo,
		}
		command.library = mockLibrary

		meta.HookGetApiByPath = func(fn func(productCode string, version string, method string, path string) (meta.Api, bool)) func(productCode string, version string, method string, path string) (meta.Api, bool) {
			return func(productCode string, version string, method string, path string) (meta.Api, bool) {
				return meta.Api{}, false // API not found
			}
		}

		ctx := cli.NewCommandContext(stdout, stderr)
		ctx.EnterCommand(cmd)
		regionflag := config.NewRegionFlag()
		regionflag.SetAssigned(true)
		regionflag.SetValue("cn-hangzhou")
		ctx.Flags().Add(regionflag)
		accessKeyIDFlag := config.NewAccessKeyIdFlag()
		accessKeyIDFlag.SetAssigned(true)
		accessKeyIDFlag.SetValue("test-access-key-id")
		ctx.Flags().Add(accessKeyIDFlag)
		accessKeySecretFlag := config.NewAccessKeySecretFlag()
		accessKeySecretFlag.SetAssigned(true)
		accessKeySecretFlag.SetValue("test-access-key-secret")
		ctx.Flags().Add(accessKeySecretFlag)
		ForceFlag(ctx.Flags()).SetAssigned(true) // force flag to true

		args := []string{"sls", "GET", "/nonexistent"}
		err := command.main(ctx, args)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "can not find api by path")
		assert.Contains(t, err.Error(), "/nonexistent")
		if errorWithTip, ok := err.(cli.ErrorWithTip); ok {
			assert.Contains(t, errorWithTip.GetTip("en"), "Please confirm if the API path exists")
		} else {
			t.Fatalf("Expected ErrorWithTip, got %T", err)
		}
	})
}

func TestMain_PluginExecution_KebabCase(t *testing.T) {
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)
	profile := config.Profile{
		Language: "en",
		RegionId: "cn-hangzhou",
	}
	command := NewCommando(w, profile)

	cmd := &cli.Command{}
	cmd.EnableUnknownFlag = true
	command.InitWithCommand(cmd)
	AddFlags(cmd.Flags())
	ctx.EnterCommand(cmd)
	ctx.Command().Short = &i18n.Text{}

	originalArgs := os.Args
	defer func() {
		os.Args = originalArgs
	}()

	t.Run("Kebab-case API name triggers plugin execution - plugin not found", func(t *testing.T) {
		testHome := t.TempDir()
		cleanup := setTestHomeDir(t, testHome)
		defer cleanup()

		writeMinimalConfigJSON(t, testHome)
		manifestPath := filepath.Join(testHome, ".aliyun", "plugins", "manifest.json")
		os.MkdirAll(filepath.Dir(manifestPath), 0755)
		os.WriteFile(manifestPath, []byte(`{"plugins":{}}`), 0644)

		os.Args = []string{"aliyun", "qqq", "describe-regions"}
		args := []string{"qqq", "describe-regions"}

		// Mock isInteractiveInput to false for CI/CD
		oldInteractive := isInteractiveInput
		isInteractiveInput = func() bool { return false }
		defer func() { isInteractiveInput = oldInteractive }()

		err := command.main(ctx, args)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "'qqq' is not a valid product")
	})

	t.Run("Kebab-case API name with multiple arguments extracts all args", func(t *testing.T) {
		testHome := t.TempDir()
		cleanup := setTestHomeDir(t, testHome)
		defer cleanup()

		writeMinimalConfigJSON(t, testHome)
		manifestPath := filepath.Join(testHome, ".aliyun", "plugins", "manifest.json")
		os.MkdirAll(filepath.Dir(manifestPath), 0755)
		os.WriteFile(manifestPath, []byte(`{"plugins":{}}`), 0644)

		os.Args = []string{"aliyun", "qqq", "describe-regions", "--region", "cn-hangzhou"}
		args := []string{"qqq", "describe-regions"}

		err := command.main(ctx, args)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "'qqq' is not a valid product")
	})

	t.Run("Non-kebab-case API name does not trigger plugin", func(t *testing.T) {
		testHome := t.TempDir()
		cleanup := setTestHomeDir(t, testHome)
		defer cleanup()

		manifestPath := filepath.Join(testHome, ".aliyun", "plugins", "manifest.json")
		os.MkdirAll(filepath.Dir(manifestPath), 0755)
		os.WriteFile(manifestPath, []byte(`{"plugins":{}}`), 0644)

		os.Args = []string{"aliyun", "ecs", "DescribeRegions"}
		args := []string{"ecs", "DescribeRegions"}

		// Execute main - should NOT trigger plugin execution
		// Instead, it should continue with normal product/API lookup
		err := command.main(ctx, args)
		if err != nil {
			assert.NotContains(t, err.Error(), "plugin")
		}
	})

	t.Run("Single argument does not trigger plugin check", func(t *testing.T) {
		testHome := t.TempDir()
		cleanup := setTestHomeDir(t, testHome)
		defer cleanup()

		manifestPath := filepath.Join(testHome, ".aliyun", "plugins", "manifest.json")
		os.MkdirAll(filepath.Dir(manifestPath), 0755)
		os.WriteFile(manifestPath, []byte(`{"plugins":{}}`), 0644)

		os.Args = []string{"aliyun", "fc"}
		args := []string{"fc"}

		err := command.main(ctx, args)
		if err != nil {
			assert.NotContains(t, err.Error(), "plugin")
		}
	})

	t.Run("Kebab-case with command not in os.Args", func(t *testing.T) {
		testHome := t.TempDir()
		cleanup := setTestHomeDir(t, testHome)
		defer cleanup()

		writeMinimalConfigJSON(t, testHome)
		manifestPath := filepath.Join(testHome, ".aliyun", "plugins", "manifest.json")
		os.MkdirAll(filepath.Dir(manifestPath), 0755)
		os.WriteFile(manifestPath, []byte(`{"plugins":{}}`), 0644)

		os.Args = []string{"aliyun", "other-command"}
		args := []string{"fc", "describe-regions"}

		// Mock isInteractiveInput to false for CI/CD
		oldInteractive := isInteractiveInput
		isInteractiveInput = func() bool { return false }
		defer func() { isInteractiveInput = oldInteractive }()

		err := command.main(ctx, args)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "'fc' is not a valid built-in product")
	})
}

// TestMain_PluginExecution_ProfileFailFast 验证 plugin 路径下 profile 校验失败时必须 fail-fast，不能 silent 吞错回退到默认 profile。
// 回归保护：历史上 commando.main 的 `if profile, err := LoadProfileWithContext(ctx); err == nil` 会把 err 吞掉，导致 `--profile xxx` 指向坏 profile 时悄悄换回默认 profile。
func TestMain_PluginExecution_ProfileFailFast(t *testing.T) {
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)
	bootProfile := config.Profile{
		Name:     "AkProfile", // 模拟 main.go 启动时已经加载的 default profile
		Language: "en",
		RegionId: "cn-hangzhou",
	}
	command := NewCommando(w, bootProfile)

	cmd := &cli.Command{}
	cmd.EnableUnknownFlag = true
	command.InitWithCommand(cmd)
	AddFlags(cmd.Flags())
	config.AddFlags(cmd.Flags())
	ctx.EnterCommand(cmd)
	ctx.Command().Short = &i18n.Text{}

	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	t.Run("auto-install path: invalid --profile fails fast (does not silently fall back)", func(t *testing.T) {
		testHome := t.TempDir()
		cleanup := setTestHomeDir(t, testHome)
		defer cleanup()

		// config.json: current=AkProfile（合法 AK），bad-oauth profile（OAuth 但缺 site_type）
		dir := filepath.Join(testHome, ".aliyun")
		os.MkdirAll(dir, 0755)
		os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{
  "current": "AkProfile",
  "profiles": [
    {"name":"AkProfile","mode":"AK","access_key_id":"ak","access_key_secret":"sk","region_id":"cn-hangzhou"},
    {"name":"bad-oauth","mode":"OAuth","region_id":"cn-hangzhou"}
  ]
}`), 0644)
		manifestPath := filepath.Join(testHome, ".aliyun", "plugins", "manifest.json")
		os.MkdirAll(filepath.Dir(manifestPath), 0755)
		os.WriteFile(manifestPath, []byte(`{"plugins":{}}`), 0644)

		// 模拟 --profile bad-oauth
		config.ProfileFlag(ctx.Flags()).SetAssigned(true)
		config.ProfileFlag(ctx.Flags()).SetValue("bad-oauth")
		defer func() { config.ProfileFlag(ctx.Flags()).SetAssigned(false) }()

		os.Args = []string{"aliyun", "qqq", "describe-regions", "--profile", "bad-oauth"}
		args := []string{"qqq", "describe-regions"}

		oldInteractive := isInteractiveInput
		isInteractiveInput = func() bool { return false }
		defer func() { isInteractiveInput = oldInteractive }()

		err := command.main(ctx, args)
		assert.Error(t, err)
		// 关键：应当报 profile 校验错误，而不是 silent 回到 AkProfile 后再报"插件没找到"
		assert.Contains(t, err.Error(), "oauth_site_type",
			"profile 校验错误必须暴露出来，不能 silent 吞掉 / 回退到 default profile")
		assert.NotContains(t, err.Error(), "is not a valid product",
			"fail-fast 应当在 profile 校验阶段就触发，不能继续走到 plugin 检查")
	})

	t.Run("execution prep path: invalid --profile fails fast when plugin already installed", func(t *testing.T) {
		testHome := t.TempDir()
		cleanup := setTestHomeDir(t, testHome)
		defer cleanup()

		dir := filepath.Join(testHome, ".aliyun")
		os.MkdirAll(dir, 0755)
		os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{
  "current": "AkProfile",
  "profiles": [
    {"name":"AkProfile","mode":"AK","access_key_id":"ak","access_key_secret":"sk","region_id":"cn-hangzhou"},
    {"name":"bad-oauth","mode":"OAuth","region_id":"cn-hangzhou"}
  ]
}`), 0644)

		pluginDir := filepath.Join(testHome, ".aliyun", "plugins")
		pluginPath := filepath.Join(pluginDir, "qqq")
		os.MkdirAll(pluginPath, 0755)
		manifestPath := filepath.Join(pluginDir, "manifest.json")
		manifest := fmt.Sprintf(`{
  "plugins": {
    "qqq": {
      "name": "qqq",
      "version": "1.0.0",
      "path": %q
    }
  }
}`, pluginPath)
		os.WriteFile(manifestPath, []byte(manifest), 0644)

		config.ProfileFlag(ctx.Flags()).SetAssigned(true)
		config.ProfileFlag(ctx.Flags()).SetValue("bad-oauth")
		defer func() { config.ProfileFlag(ctx.Flags()).SetAssigned(false) }()

		os.Args = []string{"aliyun", "qqq", "describe-regions", "--profile", "bad-oauth"}
		args := []string{"qqq", "describe-regions"}

		err := command.main(ctx, args)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "oauth_site_type",
			"执行前 LoadProfileWithContext 必须 fail-fast，不能保留启动时的 AkProfile")
		assert.NotContains(t, err.Error(), "failed to resolve plugin binary",
			"profile 错误应早于 ExecutePlugin，不应尝试拉起插件进程")
		_, ok := err.(cli.ErrorWithTip)
		assert.True(t, ok, "应返回 ErrorWithTip，与 legacy 路径一致")
		if ok {
			assert.Contains(t, err.(cli.ErrorWithTip).GetTip("en"), "aliyun configure")
		}
	})
}

// TestMain_PluginExecution_LenientProfile mirrors ProfileFailFast but for
// plugins that opt out via `profileRequired: false`. The contract:
//   - bad / missing profile must NOT abort the call before the plugin runs;
//     instead the plugin process is invoked with a baseline env.
//   - we still surface plugin-level errors (e.g. missing binary), so we
//     assert the failure point shifted from profile-validation to plugin
//     execution.
func TestMain_PluginExecution_LenientProfile(t *testing.T) {
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)
	bootProfile := config.Profile{
		Name:     "AkProfile",
		Language: "en",
		RegionId: "cn-hangzhou",
	}
	command := NewCommando(w, bootProfile)

	cmd := &cli.Command{}
	cmd.EnableUnknownFlag = true
	command.InitWithCommand(cmd)
	AddFlags(cmd.Flags())
	config.AddFlags(cmd.Flags())
	ctx.EnterCommand(cmd)
	ctx.Command().Short = &i18n.Text{}

	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	// Plugin manifest declares profileRequired=false. The bad-oauth profile
	// would normally fail Profile.Validate; lenient mode must swallow that
	// and still attempt to spawn the plugin (which then fails with a binary
	// resolution error because we don't actually drop a real binary).
	t.Run("invalid profile is bypassed when plugin opts out via profileRequired=false", func(t *testing.T) {
		testHome := t.TempDir()
		cleanup := setTestHomeDir(t, testHome)
		defer cleanup()

		dir := filepath.Join(testHome, ".aliyun")
		os.MkdirAll(dir, 0755)
		os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{
  "current": "AkProfile",
  "profiles": [
    {"name":"AkProfile","mode":"AK","access_key_id":"ak","access_key_secret":"sk","region_id":"cn-hangzhou"},
    {"name":"bad-oauth","mode":"OAuth","region_id":"cn-hangzhou"}
  ]
}`), 0644)

		pluginDir := filepath.Join(testHome, ".aliyun", "plugins")
		pluginPath := filepath.Join(pluginDir, "rdc")
		os.MkdirAll(pluginPath, 0755)
		manifestPath := filepath.Join(pluginDir, "manifest.json")
		manifest := fmt.Sprintf(`{
  "plugins": {
    "rdc": {
      "name": "rdc",
      "version": "1.0.0",
      "path": %q,
      "profileRequired": false
    }
  }
}`, pluginPath)
		os.WriteFile(manifestPath, []byte(manifest), 0644)

		config.ProfileFlag(ctx.Flags()).SetAssigned(true)
		config.ProfileFlag(ctx.Flags()).SetValue("bad-oauth")
		defer func() { config.ProfileFlag(ctx.Flags()).SetAssigned(false) }()

		os.Args = []string{"aliyun", "rdc", "list", "--profile", "bad-oauth"}
		args := []string{"rdc", "list"}

		err := command.main(ctx, args)
		assert.Error(t, err)
		assert.NotContains(t, err.Error(), "oauth_site_type",
			"profileRequired=false 的插件不应被 profile 校验阻塞")
		assert.Contains(t, err.Error(), "failed to resolve plugin binary",
			"宽松路径下应继续到 ExecutePlugin，错误来自 binary 解析阶段")
	})

	// Even when no profile exists at all (no config.json on disk),
	// profileRequired=false plugins must still execute end-to-end up to the
	// plugin binary stage.
	t.Run("missing profile config does not block opted-out plugin", func(t *testing.T) {
		testHome := t.TempDir()
		cleanup := setTestHomeDir(t, testHome)
		defer cleanup()

		// Reset profile flag possibly set by previous subtests.
		config.ProfileFlag(ctx.Flags()).SetAssigned(false)
		config.ProfileFlag(ctx.Flags()).SetValue("")

		// Intentionally do NOT write config.json — profile loading will
		// either return an empty profile or fail; either way, lenient mode
		// must keep going.
		pluginDir := filepath.Join(testHome, ".aliyun", "plugins")
		pluginPath := filepath.Join(pluginDir, "rdc")
		os.MkdirAll(pluginPath, 0755)
		manifestPath := filepath.Join(pluginDir, "manifest.json")
		manifest := fmt.Sprintf(`{
  "plugins": {
    "rdc": {
      "name": "rdc",
      "version": "1.0.0",
      "path": %q,
      "profileRequired": false
    }
  }
}`, pluginPath)
		os.WriteFile(manifestPath, []byte(manifest), 0644)

		os.Args = []string{"aliyun", "rdc", "list"}
		args := []string{"rdc", "list"}

		err := command.main(ctx, args)
		assert.Error(t, err)
		assert.NotContains(t, err.Error(), "profile not found",
			"profileRequired=false 不应触发 'profile not found' 报错")
		assert.NotContains(t, err.Error(), "Configuration failed",
			"profileRequired=false 不应触发 Configuration failed 报错")
		assert.Contains(t, err.Error(), "failed to resolve plugin binary",
			"宽松路径下错误应来自 binary 解析阶段")
	})
}

func TestPluginExecutionLogic(t *testing.T) {
	// Setup test environment
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)
	profile := config.Profile{
		Language: "en",
		RegionId: "cn-hangzhou",
	}
	command := NewCommando(stdout, profile)

	cmd := &cli.Command{}
	cmd.EnableUnknownFlag = true
	command.InitWithCommand(cmd)
	AddFlags(cmd.Flags())
	ctx.EnterCommand(cmd)
	regionflag := config.NewRegionFlag()
	regionflag.SetAssigned(true)
	regionflag.SetValue("cn-hangzhou")
	ctx.Flags().Add(regionflag)
	accessKeyIDFlag := config.NewAccessKeyIdFlag()
	accessKeyIDFlag.SetAssigned(true)
	accessKeyIDFlag.SetValue("test-access-key-id")
	ctx.Flags().Add(accessKeyIDFlag)
	accessKeySecretFlag := config.NewAccessKeySecretFlag()
	accessKeySecretFlag.SetAssigned(true)
	accessKeySecretFlag.SetValue("test-access-key-secret")
	ctx.Flags().Add(accessKeySecretFlag)

	ctx.Command().Short = &i18n.Text{}

	originalArgs := os.Args
	defer func() {
		os.Args = originalArgs
	}()

	// Set up plugin environment
	testHome := t.TempDir()
	cleanup := setTestHomeDir(t, testHome)
	defer cleanup()

	originalIgnoreProfile := os.Getenv("ALIBABA_CLOUD_IGNORE_PROFILE")
	os.Setenv("ALIBABA_CLOUD_IGNORE_PROFILE", "TRUE")
	defer func() {
		if originalIgnoreProfile == "" {
			os.Unsetenv("ALIBABA_CLOUD_IGNORE_PROFILE")
		} else {
			os.Setenv("ALIBABA_CLOUD_IGNORE_PROFILE", originalIgnoreProfile)
		}
	}()

	pluginDir := filepath.Join(testHome, ".aliyun", "plugins")
	manifestPath := filepath.Join(pluginDir, "manifest.json")
	err := os.MkdirAll(pluginDir, 0755)
	assert.NoError(t, err)

	t.Run("Plugin execution with valid executable", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("shell script test skipped on Windows")
		}
		testPluginManifest := `{
			"plugins": {
				"testplugin": {
					"name": "testplugin",
					"version": "1.0.0",
					"path": "` + filepath.Join(pluginDir, "testplugin") + `"
				}
			}
		}`
		err := os.WriteFile(manifestPath, []byte(testPluginManifest), 0644)
		assert.NoError(t, err)

		// Create mock plugin executable that succeeds
		pluginPath := filepath.Join(pluginDir, "testplugin", "testplugin")
		err = os.MkdirAll(filepath.Dir(pluginPath), 0755)
		assert.NoError(t, err)

		mockPluginScript := `#!/bin/bash
echo "Plugin executed successfully"
exit 0
`
		err = os.WriteFile(pluginPath, []byte(mockPluginScript), 0755)
		assert.NoError(t, err)

		os.Args = []string{"aliyun", "testplugin", "test-command"}
		args := []string{"testplugin", "test-command"}

		err = command.main(ctx, args)
		assert.NoError(t, err)
		assert.Contains(t, stdout.String(), "Plugin executed successfully")
	})

	t.Run("Plugin binary not found after installation check", func(t *testing.T) {
		// Test case where plugin is in manifest but binary doesn't exist
		// This tests the (ok=false, err=nil) path from ExecutePlugin
		testPluginManifest := `{
			"plugins": {
				"missingplugin": {
					"name": "missingplugin",
					"version": "1.0.0",
					"path": "` + filepath.ToSlash(filepath.Join(pluginDir, "missingplugin")) + `",
					"command": "missing-cmd"
				}
			}
		}`
		err := os.WriteFile(manifestPath, []byte(testPluginManifest), 0644)
		assert.NoError(t, err)

		// Create directory but don't create the binary
		err = os.MkdirAll(filepath.Join(pluginDir, "missingplugin"), 0755)
		assert.NoError(t, err)

		os.Args = []string{"aliyun", "missingplugin", "missing-cmd"}
		args := []string{"missingplugin", "missing-cmd"}

		err = command.main(ctx, args)
		// Should get "plugin not found" error
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to resolve plugin binary path")
		assert.Contains(t, err.Error(), "plugin")
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("Environment variable ALIBABA_CLOUD_ORIGINAL_PRODUCT_HELP skips plugin for single arg", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("shell script test skipped on Windows")
		}
		// Set environment variable
		originalEnv := os.Getenv("ALIBABA_CLOUD_ORIGINAL_PRODUCT_HELP")
		os.Setenv("ALIBABA_CLOUD_ORIGINAL_PRODUCT_HELP", "true")
		defer func() {
			if originalEnv == "" {
				os.Unsetenv("ALIBABA_CLOUD_ORIGINAL_PRODUCT_HELP")
			} else {
				os.Setenv("ALIBABA_CLOUD_ORIGINAL_PRODUCT_HELP", originalEnv)
			}
		}()

		// Create a plugin that exists
		testPluginManifest := `{
			"plugins": {
				"envtest": {
					"name": "envtest",
					"version": "1.0.0",
					"path": "` + filepath.Join(pluginDir, "envtest") + `",
					"command": "envtest"
				}
			}
		}`
		err := os.WriteFile(manifestPath, []byte(testPluginManifest), 0644)
		assert.NoError(t, err)

		// Create the plugin binary so it could be executed
		pluginPath := filepath.Join(pluginDir, "envtest", "envtest")
		err = os.MkdirAll(filepath.Dir(pluginPath), 0755)
		assert.NoError(t, err)

		mockPluginScript := `#!/bin/bash
echo "Plugin should not execute"
exit 0
`
		err = os.WriteFile(pluginPath, []byte(mockPluginScript), 0755)
		assert.NoError(t, err)

		// Test with single argument (product-level help scenario)
		os.Args = []string{"aliyun", "envtest"}
		args := []string{"envtest"}

		stdout.Reset()
		err = command.main(ctx, args)

		// With ALIBABA_CLOUD_ORIGINAL_PRODUCT_HELP=true and single arg,
		// plugin execution is skipped
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "not a valid command or product")
		assert.NotContains(t, stdout.String(), "Plugin should not execute")

	})

	t.Run("Kebab-case command triggers plugin execution", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("shell script test skipped on Windows")
		}
		// Verify that kebab-case in second argument triggers plugin execution
		testPluginManifest := `{
			"plugins": {
				"kebabtest": {
					"name": "kebabtest",
					"version": "1.0.0",
					"path": "` + filepath.Join(pluginDir, "kebabtest") + `",
					"command": "kebabtest"
				}
			}
		}`
		err := os.WriteFile(manifestPath, []byte(testPluginManifest), 0644)
		assert.NoError(t, err)

		pluginPath := filepath.Join(pluginDir, "kebabtest", "kebabtest")
		err = os.MkdirAll(filepath.Dir(pluginPath), 0755)
		assert.NoError(t, err)

		mockPluginScript := `#!/bin/bash
echo "Kebab-case command executed"
exit 0
`
		err = os.WriteFile(pluginPath, []byte(mockPluginScript), 0755)
		assert.NoError(t, err)

		os.Args = []string{"aliyun", "kebabtest", "list-resources"}
		args := []string{"kebabtest", "list-resources"}

		stdout.Reset()
		err = command.main(ctx, args)

		// Should execute plugin due to kebab-case in args[1]
		assert.NoError(t, err)
		assert.Contains(t, stdout.String(), "Kebab-case command executed")
	})
}

func TestSingleProductPluginExecution(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)
	profile := config.Profile{
		Language: "en",
		RegionId: "cn-hangzhou",
	}
	command := NewCommando(stdout, profile)

	cmd := &cli.Command{}
	cmd.EnableUnknownFlag = true
	command.InitWithCommand(cmd)
	AddFlags(cmd.Flags())
	ctx.EnterCommand(cmd)

	regionflag := config.NewRegionFlag()
	regionflag.SetAssigned(true)
	regionflag.SetValue("cn-hangzhou")
	ctx.Flags().Add(regionflag)
	accessKeyIDFlag := config.NewAccessKeyIdFlag()
	accessKeyIDFlag.SetAssigned(true)
	accessKeyIDFlag.SetValue("test-access-key-id")
	ctx.Flags().Add(accessKeyIDFlag)
	accessKeySecretFlag := config.NewAccessKeySecretFlag()
	accessKeySecretFlag.SetAssigned(true)
	accessKeySecretFlag.SetValue("test-access-key-secret")
	ctx.Flags().Add(accessKeySecretFlag)

	ctx.Command().Short = &i18n.Text{}

	originalArgs := os.Args
	defer func() {
		os.Args = originalArgs
	}()

	testHome := t.TempDir()
	cleanup := setTestHomeDir(t, testHome)
	defer cleanup()

	originalIgnoreProfile := os.Getenv("ALIBABA_CLOUD_IGNORE_PROFILE")
	os.Setenv("ALIBABA_CLOUD_IGNORE_PROFILE", "TRUE")
	defer func() {
		if originalIgnoreProfile == "" {
			os.Unsetenv("ALIBABA_CLOUD_IGNORE_PROFILE")
		} else {
			os.Setenv("ALIBABA_CLOUD_IGNORE_PROFILE", originalIgnoreProfile)
		}
	}()

	pluginDir := filepath.Join(testHome, ".aliyun", "plugins")
	manifestPath := filepath.Join(pluginDir, "manifest.json")
	err := os.MkdirAll(pluginDir, 0755)
	assert.NoError(t, err)

	t.Run("Single arg - plugin installed and executes successfully", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("shell script test skipped on Windows")
		}
		// installed=true, ok=true, err=nil -> return nil
		testPluginManifest := `{
			"plugins": {
				"singletest": {
					"name": "singletest",
					"version": "1.0.0",
					"path": "` + filepath.Join(pluginDir, "singletest") + `",
					"command": "singletest"
				}
			}
		}`
		err := os.WriteFile(manifestPath, []byte(testPluginManifest), 0644)
		assert.NoError(t, err)

		pluginPath := filepath.Join(pluginDir, "singletest", "singletest")
		err = os.MkdirAll(filepath.Dir(pluginPath), 0755)
		assert.NoError(t, err)

		mockPluginScript := `#!/bin/bash
echo "Single product plugin executed"
exit 0
`
		err = os.WriteFile(pluginPath, []byte(mockPluginScript), 0755)
		assert.NoError(t, err)

		os.Args = []string{"aliyun", "singletest"}
		args := []string{"singletest"}

		stdout.Reset()
		err = command.main(ctx, args)

		assert.NoError(t, err)
		assert.Contains(t, stdout.String(), "Single product plugin executed")
	})

	t.Run("Single arg - plugin installed but binary not found", func(t *testing.T) {
		// installed=true, ok=false -> return error
		testPluginManifest := `{
			"plugins": {
				"missingbin": {
					"name": "missingbin",
					"version": "1.0.0",
					"path": "` + filepath.ToSlash(filepath.Join(pluginDir, "missingbin")) + `",
					"command": "missingbin"
				}
			}
		}`
		err := os.WriteFile(manifestPath, []byte(testPluginManifest), 0644)
		assert.NoError(t, err)

		// Create directory but don't create the binary
		err = os.MkdirAll(filepath.Join(pluginDir, "missingbin"), 0755)
		assert.NoError(t, err)

		os.Args = []string{"aliyun", "missingbin"}
		args := []string{"missingbin"}

		stdout.Reset()
		err = command.main(ctx, args)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "plugin")
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("Single arg - plugin not installed", func(t *testing.T) {
		// installed=false -> continue to normal flow
		testPluginManifest := `{
			"plugins": {}
		}`
		err := os.WriteFile(manifestPath, []byte(testPluginManifest), 0644)
		assert.NoError(t, err)

		os.Args = []string{"aliyun", "notinstalled"}
		args := []string{"notinstalled"}

		stdout.Reset()
		err = command.main(ctx, args)

		// Plugin not installed, should continue to normal product lookup
		assert.Error(t, err)
		assert.NotContains(t, err.Error(), "plugin notinstalled not found")
	})

	t.Run("Single arg - IsPluginInstalled returns error", func(t *testing.T) {
		// err != nil from IsPluginInstalled

		// Write invalid JSON to manifest
		err := os.WriteFile(manifestPath, []byte(`{invalid json`), 0644)
		assert.NoError(t, err)

		os.Args = []string{"aliyun", "testprod"}
		args := []string{"testprod"}

		stdout.Reset()
		err = command.main(ctx, args)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to check plugin status")
	})

	t.Run("Single arg - with ALIBABA_CLOUD_ORIGINAL_PRODUCT_HELP=true", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("shell script test skipped on Windows")
		}
		// Test that environment variable skips the entire plugin check block
		originalEnv := os.Getenv("ALIBABA_CLOUD_ORIGINAL_PRODUCT_HELP")
		os.Setenv("ALIBABA_CLOUD_ORIGINAL_PRODUCT_HELP", "true")
		defer func() {
			if originalEnv == "" {
				os.Unsetenv("ALIBABA_CLOUD_ORIGINAL_PRODUCT_HELP")
			} else {
				os.Setenv("ALIBABA_CLOUD_ORIGINAL_PRODUCT_HELP", originalEnv)
			}
		}()

		testPluginManifest := `{
			"plugins": {
				"envskip": {
					"name": "envskip",
					"version": "1.0.0",
					"path": "` + filepath.Join(pluginDir, "envskip") + `",
					"command": "envskip"
				}
			}
		}`
		err := os.WriteFile(manifestPath, []byte(testPluginManifest), 0644)
		assert.NoError(t, err)

		pluginPath := filepath.Join(pluginDir, "envskip", "envskip")
		err = os.MkdirAll(filepath.Dir(pluginPath), 0755)
		assert.NoError(t, err)

		mockPluginScript := `#!/bin/bash
echo "Should not execute"
exit 0
`
		err = os.WriteFile(pluginPath, []byte(mockPluginScript), 0755)
		assert.NoError(t, err)

		os.Args = []string{"aliyun", "envskip"}
		args := []string{"envskip"}

		stdout.Reset()
		err = command.main(ctx, args)

		// With ALIBABA_CLOUD_ORIGINAL_PRODUCT_HELP=true, the entire if block is skipped
		// Plugin should NOT be executed
		assert.NotContains(t, stdout.String(), "Should not execute")

		// Should try to process as normal product command
		assert.Error(t, err)
	})
}

// TestAutoInstallPlugin tests the autoInstallPlugin function
func TestAutoInstallPlugin(t *testing.T) {
	t.Run("Install_fails_-_plugin_not_in_index", func(t *testing.T) {
		testHome := t.TempDir()
		cleanup := setTestHomeDir(t, testHome)
		defer cleanup()

		// Create minimal plugin infrastructure
		pluginDir := filepath.Join(testHome, ".aliyun", "plugins")
		os.MkdirAll(pluginDir, 0755)

		// Create empty manifest
		manifestPath := filepath.Join(pluginDir, "manifest.json")
		manifestJSON := `{"plugins":{}}`
		os.WriteFile(manifestPath, []byte(manifestJSON), 0644)

		stdout := new(bytes.Buffer)
		stderr := new(bytes.Buffer)
		ctx := cli.NewCommandContext(stdout, stderr)
		profile := config.Profile{
			Language: "en",
			RegionId: "cn-hangzhou",
		}
		command := NewCommando(stdout, profile)

		mgr, err := plugin.NewManager()
		assert.NoError(t, err)

		pluginName, err := command.autoInstallPlugin(ctx, mgr, "nonexistent-plugin", "some-command", false)

		assert.Error(t, err, "Should fail when plugin is not in index")
		assert.Empty(t, pluginName, "Plugin name should be empty on failure")
		assert.Contains(t, err.Error(), "failed to install plugin")

		// Verify stderr output
		stderrOutput := stderr.String()
		assert.Contains(t, stderrOutput, "Plugin 'nonexistent-plugin' is required for command 'some-command'")
		assert.Contains(t, stderrOutput, "Auto-installing plugin 'nonexistent-plugin'...")
		assert.NotContains(t, stderrOutput, "including pre-release versions", "Should not mention pre-release when enablePre=false")
	})

	t.Run("Install_fails_-_plugin_not_in_index_with_enablePre", func(t *testing.T) {
		testHome := t.TempDir()
		cleanup := setTestHomeDir(t, testHome)
		defer cleanup()

		pluginDir := filepath.Join(testHome, ".aliyun", "plugins")
		os.MkdirAll(pluginDir, 0755)

		manifestPath := filepath.Join(pluginDir, "manifest.json")
		manifestJSON := `{"plugins":{}}`
		os.WriteFile(manifestPath, []byte(manifestJSON), 0644)

		stdout := new(bytes.Buffer)
		stderr := new(bytes.Buffer)
		ctx := cli.NewCommandContext(stdout, stderr)
		profile := config.Profile{
			Language: "en",
			RegionId: "cn-hangzhou",
		}
		command := NewCommando(stdout, profile)

		mgr, err := plugin.NewManager()
		assert.NoError(t, err)

		pluginName, err := command.autoInstallPlugin(ctx, mgr, "nonexistent-plugin", "some-command", true)

		assert.Error(t, err, "Should fail when plugin is not in index")
		assert.Empty(t, pluginName, "Plugin name should be empty on failure")

		stderrOutput := stderr.String()
		assert.Contains(t, stderrOutput, "Plugin 'nonexistent-plugin' is required for command 'some-command'")
		assert.Contains(t, stderrOutput, "Auto-installing plugin 'nonexistent-plugin' (including pre-release versions)...")
	})

	t.Run("Output_format_verification", func(t *testing.T) {
		testHome := t.TempDir()
		cleanup := setTestHomeDir(t, testHome)
		defer cleanup()

		pluginDir := filepath.Join(testHome, ".aliyun", "plugins")
		os.MkdirAll(pluginDir, 0755)

		manifestPath := filepath.Join(pluginDir, "manifest.json")
		manifestJSON := `{"plugins":{}}`
		os.WriteFile(manifestPath, []byte(manifestJSON), 0644)

		stdout := new(bytes.Buffer)
		stderr := new(bytes.Buffer)
		ctx := cli.NewCommandContext(stdout, stderr)
		profile := config.Profile{
			Language: "en",
			RegionId: "cn-hangzhou",
		}
		command := NewCommando(stdout, profile)

		mgr, err := plugin.NewManager()
		assert.NoError(t, err)

		command.autoInstallPlugin(ctx, mgr, "test-plugin", "test-command", false)

		stderrOutput := stderr.String()
		assert.Contains(t, stderrOutput, "Plugin 'test-plugin' is required for command 'test-command' but not installed.")
		assert.Contains(t, stderrOutput, "Auto-installing plugin 'test-plugin'...")

		stderr.Reset()

		command.autoInstallPlugin(ctx, mgr, "test-plugin", "test-command", true)

		stderrOutput = stderr.String()
		assert.Contains(t, stderrOutput, "Plugin 'test-plugin' is required for command 'test-command' but not installed.")
		assert.Contains(t, stderrOutput, "Auto-installing plugin 'test-plugin' (including pre-release versions)...")
	})

	t.Run("Manager_creation_validation", func(t *testing.T) {
		testHome := t.TempDir()
		cleanup := setTestHomeDir(t, testHome)
		defer cleanup()

		pluginDir := filepath.Join(testHome, ".aliyun", "plugins")
		os.MkdirAll(pluginDir, 0755)

		stdout := new(bytes.Buffer)
		stderr := new(bytes.Buffer)
		ctx := cli.NewCommandContext(stdout, stderr)
		profile := config.Profile{
			Language: "en",
			RegionId: "cn-hangzhou",
		}
		command := NewCommando(stdout, profile)

		mgr, err := plugin.NewManager()
		assert.NoError(t, err)
		assert.NotNil(t, mgr, "Manager should be created successfully")

		_, err = command.autoInstallPlugin(ctx, mgr, "test-plugin", "test-command", false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to install plugin")
	})
}

func TestHandleInstallError(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)
	profile := config.Profile{
		Language: "en",
	}
	command := NewCommando(stdout, profile)

	t.Run("Stable version not available error", func(t *testing.T) {
		stderr.Reset()
		err := fmt.Errorf("no stable version available for plugin test-plugin")
		command.handleInstallError(ctx, err, "test-plugin", false)

		output := stderr.String()
		assert.Contains(t, output, "no stable version available for plugin test-plugin")
		assert.Contains(t, output, "Tip: This command may require a pre-release version")
		assert.Contains(t, output, "aliyun configure set --auto-plugin-install-enable-pre true")
		assert.Contains(t, output, "aliyun plugin install --names test-plugin --enable-pre")
	})

	t.Run("Stable version not available error with enablePre=true", func(t *testing.T) {
		stderr.Reset()
		err := fmt.Errorf("no stable version available for plugin test-plugin")
		command.handleInstallError(ctx, err, "test-plugin", true)

		output := stderr.String()
		assert.Empty(t, output, "Should not print tip when enablePre is already true")
	})

	t.Run("Other error", func(t *testing.T) {
		stderr.Reset()
		err := fmt.Errorf("some other error")
		command.handleInstallError(ctx, err, "test-plugin", false)

		output := stderr.String()
		assert.Empty(t, output, "Should not print tip for unrelated errors")
	})
}

func TestInteractiveInstallPlugin(t *testing.T) {
	testHome := t.TempDir()
	cleanup := setTestHomeDir(t, testHome)
	defer cleanup()

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)
	profile := config.Profile{Language: "en"}
	command := NewCommando(stdout, profile)

	mgr, err := plugin.NewManager()
	assert.NoError(t, err)

	originalStdin := stdin
	defer func() { stdin = originalStdin }()

	t.Run("User accepts installation", func(t *testing.T) {
		stderr.Reset()
		stdin = strings.NewReader("y\n")
		// expect this to fail eventually because the plugin doesn't exist in index,
		// but we want to check if it processed the "y" correctly.
		pluginName, err := command.interactiveInstallPlugin(ctx, mgr, "test-plugin", "test-command", false)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to install plugin 'test-plugin'")
		assert.Empty(t, pluginName)

		output := stderr.String()
		assert.Contains(t, output, "Do you want to install it? [Y/n]:")
		assert.Contains(t, output, "Installing plugin 'test-plugin'...")
	})

	t.Run("User cancels installation", func(t *testing.T) {
		stderr.Reset()
		stdin = strings.NewReader("n\n")
		pluginName, err := command.interactiveInstallPlugin(ctx, mgr, "test-plugin", "test-command", false)

		assert.NoError(t, err)
		assert.Empty(t, pluginName)

		output := stderr.String()
		assert.Contains(t, output, "Do you want to install it? [Y/n]:")
		assert.Contains(t, output, "Installation cancelled.")
	})

	t.Run("User accepts with 'yes'", func(t *testing.T) {
		stderr.Reset()
		stdin = strings.NewReader("YES\n")
		_, err := command.interactiveInstallPlugin(ctx, mgr, "test-plugin", "test-command", false)
		assert.Error(t, err) // Should proceed to install and fail
		assert.Contains(t, stderr.String(), "Installing plugin 'test-plugin'...")
	})

	t.Run("Default to yes on empty input", func(t *testing.T) {
		stderr.Reset()
		stdin = strings.NewReader("\n")
		_, err := command.interactiveInstallPlugin(ctx, mgr, "test-plugin", "test-command", false)
		assert.Error(t, err) // Should proceed to install and fail
		assert.Contains(t, stderr.String(), "Installing plugin 'test-plugin'...")
	})
}

// writeMinimalConfigJSON 在 testHome 里写入一份最小合法 config.json，让
// LoadProfileWithContext / profile.Validate 能正常通过，便于测试聚焦在 plugin
// 路径本身的行为（不让"region/mode 缺失"的校验错误把测试干扰掉）。
func writeMinimalConfigJSON(t *testing.T, testHome string) {
	t.Helper()
	dir := filepath.Join(testHome, ".aliyun")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	content := `{
  "current": "default",
  "profiles": [
    {
      "name": "default",
      "mode": "AK",
      "access_key_id": "test-ak",
      "access_key_secret": "test-sk",
      "region_id": "cn-hangzhou",
      "language": "en"
    }
  ]
}`
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(content), 0644); err != nil {
		t.Fatalf("write config.json: %v", err)
	}
}

// 构造一个带 --language flag 注册的 ctx，便于测试 setLangEnv 在不同 flag / profile 组合下选用哪个语言。
// 注意：ctx.EnterCommand 必须在 AddFlags 之前调用，否则 ctx.Flags() 返回的是命令级
// 子 FlagSet，root 上注册的 language flag 读不到。
func newCtxWithLangFlag(t *testing.T, langFlag string, flagAssigned bool) *cli.Context {
	t.Helper()
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)
	ctx.EnterCommand(&cli.Command{})
	config.AddFlags(ctx.Flags())
	if flagAssigned {
		config.LanguageFlag(ctx.Flags()).SetAssigned(true)
		config.LanguageFlag(ctx.Flags()).SetValue(langFlag)
	}
	return ctx
}

func TestSetLangEnv(t *testing.T) {
	t.Run("nil ctx is noop", func(t *testing.T) {
		c := &Commando{}
		assert.NotPanics(t, func() { c.setLangEnv(nil) })
	})

	t.Run("--language flag wins over profile", func(t *testing.T) {
		ctx := newCtxWithLangFlag(t, "en", true)
		c := &Commando{}
		c.profile.Language = "zh" // profile 说 zh，flag 说 en
		c.setLangEnv(ctx)
		assert.Equal(t, "en_US.UTF-8", ctx.GetRuntimeEnvs()["LANG"])
	})

	t.Run("--language flag wins when profile empty", func(t *testing.T) {
		ctx := newCtxWithLangFlag(t, "en", true)
		c := &Commando{} // profile.Language == ""
		c.setLangEnv(ctx)
		assert.Equal(t, "en_US.UTF-8", ctx.GetRuntimeEnvs()["LANG"])
	})

	t.Run("--language flag zh", func(t *testing.T) {
		ctx := newCtxWithLangFlag(t, "zh", true)
		c := &Commando{}
		c.setLangEnv(ctx)
		assert.Equal(t, "zh_CN.UTF-8", ctx.GetRuntimeEnvs()["LANG"])
	})

	t.Run("falls back to profile when flag not assigned", func(t *testing.T) {
		ctx := newCtxWithLangFlag(t, "", false)
		c := &Commando{}
		c.profile.Language = "zh"
		c.setLangEnv(ctx)
		assert.Equal(t, "zh_CN.UTF-8", ctx.GetRuntimeEnvs()["LANG"])
	})

	t.Run("falls back to i18n.GetLanguage when both empty", func(t *testing.T) {
		prev := i18n.GetLanguage()
		t.Cleanup(func() { i18n.SetLanguage(prev) })
		i18n.SetLanguage("en")

		ctx := newCtxWithLangFlag(t, "", false)
		c := &Commando{} // profile.Language == ""
		c.setLangEnv(ctx)
		assert.Equal(t, "en_US.UTF-8", ctx.GetRuntimeEnvs()["LANG"])
	})

	t.Run("unknown language defaults to en_US.UTF-8", func(t *testing.T) {
		ctx := newCtxWithLangFlag(t, "fr", true)
		c := &Commando{}
		c.setLangEnv(ctx)
		assert.Equal(t, "en_US.UTF-8", ctx.GetRuntimeEnvs()["LANG"])
	})

	t.Run("preserves existing runtime envs", func(t *testing.T) {
		ctx := newCtxWithLangFlag(t, "en", true)
		ctx.SetRuntimeEnvs(map[string]string{"FOO": "bar"})
		c := &Commando{}
		c.setLangEnv(ctx)
		envs := ctx.GetRuntimeEnvs()
		assert.Equal(t, "en_US.UTF-8", envs["LANG"])
		assert.Equal(t, "bar", envs["FOO"], "existing entries must be kept")
	})
}

// writeSafetyPolicy writes a safety-policy.json under ~/.aliyun in testHome and
// makes sure no leftover env override is present, so the policy on disk is the
// one that actually takes effect during the test.
func writeSafetyPolicy(t *testing.T, testHome string, rules []safety.Rule) {
	t.Helper()

	for _, k := range []string{safety.EnvSafetyPolicyEnabled, safety.EnvSafetyPolicyRules} {
		prev, had := os.LookupEnv(k)
		os.Unsetenv(k)
		t.Cleanup(func() {
			if had {
				os.Setenv(k, prev)
			} else {
				os.Unsetenv(k)
			}
		})
	}

	dir := filepath.Join(testHome, ".aliyun")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	if err := safety.SavePolicy(dir, &safety.Policy{Enabled: true, Rules: rules}); err != nil {
		t.Fatalf("save safety policy: %v", err)
	}
}

// newSafetyCommandoTestCtx builds a ctx + commando wired with the standard
// flags so command.main can run end-to-end. Mirrors the setup used by other
// integration tests in this file.
func newSafetyCommandoTestCtx(t *testing.T) (*cli.Context, *Commando) {
	t.Helper()
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)
	profile := config.Profile{
		Language:        "en",
		Mode:            "AK",
		AccessKeyId:     "test-ak",
		AccessKeySecret: "test-sk",
		RegionId:        "cn-hangzhou",
	}
	command := NewCommando(stdout, profile)

	cmd := &cli.Command{}
	cmd.EnableUnknownFlag = true
	command.InitWithCommand(cmd)
	AddFlags(cmd.Flags())
	config.AddFlags(cmd.Flags())
	ctx.EnterCommand(cmd)
	ctx.Command().Short = &i18n.Text{}
	return ctx, command
}

// TestMain_SafetyPolicyEnforcement guards the three safety-policy call sites
// in commando.main: the plugin dispatch path, the 2-arg RPC/REST-by-ApiName
// path and the 3-arg REST-by-method/path path. Each subtest writes a deny
// rule shaped exactly like what users type and asserts that command.main
// short-circuits with the canonical "blocked by safety policy" error before
// any actual API / plugin execution kicks in.
func TestMain_SafetyPolicyEnforcement(t *testing.T) {
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	t.Run("2-arg RPC: rule on ApiName blocks the call", func(t *testing.T) {
		testHome := t.TempDir()
		cleanup := setTestHomeDir(t, testHome)
		defer cleanup()
		writeMinimalConfigJSON(t, testHome)
		writeSafetyPolicy(t, testHome, []safety.Rule{
			{Pattern: "ecs:DeleteInstance", Action: safety.ActionDeny},
		})

		ctx, command := newSafetyCommandoTestCtx(t)
		os.Args = []string{"aliyun", "ecs", "DeleteInstance"}
		args := []string{"ecs", "DeleteInstance"}

		err := command.main(ctx, args)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "blocked by safety policy")
		assert.Contains(t, err.Error(), "ecs:DeleteInstance",
			"matched rule pattern should be reported back")
	})

	t.Run("2-arg ROA invoked by ApiName: rule on ApiName blocks the call", func(t *testing.T) {
		// Regression for the original bug: `aliyun sls ListProject` is
		// dispatched as REST GET / under the hood, but a rule keyed on the
		// ApiName the user actually typed must still match.
		testHome := t.TempDir()
		cleanup := setTestHomeDir(t, testHome)
		defer cleanup()
		writeMinimalConfigJSON(t, testHome)
		writeSafetyPolicy(t, testHome, []safety.Rule{
			{Pattern: "sls:ListProject", Action: safety.ActionDeny},
		})

		ctx, command := newSafetyCommandoTestCtx(t)
		os.Args = []string{"aliyun", "sls", "ListProject"}
		args := []string{"sls", "ListProject"}

		err := command.main(ctx, args)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "blocked by safety policy")
		assert.Contains(t, err.Error(), "sls:ListProject")
	})

	t.Run("3-arg REST: rule on METHOD/path blocks the call", func(t *testing.T) {
		testHome := t.TempDir()
		cleanup := setTestHomeDir(t, testHome)
		defer cleanup()
		writeMinimalConfigJSON(t, testHome)
		writeSafetyPolicy(t, testHome, []safety.Rule{
			{Pattern: "cs:DELETE/*", Action: safety.ActionDeny},
		})

		ctx, command := newSafetyCommandoTestCtx(t)
		ForceFlag(ctx.Flags()).SetAssigned(true) // skip metadata lookup for path
		os.Args = []string{"aliyun", "cs", "DELETE", "/clusters/abc"}
		args := []string{"cs", "DELETE", "/clusters/abc"}

		err := command.main(ctx, args)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "blocked by safety policy")
		assert.Contains(t, err.Error(), "cs:DELETE/*")
	})

	t.Run("plugin multi-segment: colon-joined rule blocks the call", func(t *testing.T) {
		testHome := t.TempDir()
		cleanup := setTestHomeDir(t, testHome)
		defer cleanup()
		writeMinimalConfigJSON(t, testHome)
		writeSafetyPolicy(t, testHome, []safety.Rule{
			{Pattern: "fc:function:create", Action: safety.ActionDeny},
		})

		// Pretend the plugin is installed so commando.main proceeds past the
		// "is plugin installed?" gate and reaches the safety check.
		pluginDir := filepath.Join(testHome, ".aliyun", "plugins")
		pluginPath := filepath.Join(pluginDir, "fc")
		if err := os.MkdirAll(pluginPath, 0755); err != nil {
			t.Fatalf("mkdir plugin dir: %v", err)
		}
		manifest := fmt.Sprintf(`{
  "plugins": {
    "fc": {
      "name": "fc",
      "version": "1.0.0",
      "path": %q
    }
  }
}`, pluginPath)
		if err := os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(manifest), 0644); err != nil {
			t.Fatalf("write manifest: %v", err)
		}

		ctx, command := newSafetyCommandoTestCtx(t)
		os.Args = []string{"aliyun", "fc", "function", "create"}
		args := []string{"fc", "function", "create"}

		err := command.main(ctx, args)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "blocked by safety policy")
		assert.Contains(t, err.Error(), "fc:function:create")
	})

	t.Run("plugin single-segment: rule on sub-command blocks the call", func(t *testing.T) {
		// Single-segment plugins (`aliyun fc create-function`) must keep
		// matching the historical `product:sub-command` form.
		testHome := t.TempDir()
		cleanup := setTestHomeDir(t, testHome)
		defer cleanup()
		writeMinimalConfigJSON(t, testHome)
		writeSafetyPolicy(t, testHome, []safety.Rule{
			{Pattern: "fc:create-function", Action: safety.ActionDeny},
		})

		pluginDir := filepath.Join(testHome, ".aliyun", "plugins")
		pluginPath := filepath.Join(pluginDir, "fc")
		if err := os.MkdirAll(pluginPath, 0755); err != nil {
			t.Fatalf("mkdir plugin dir: %v", err)
		}
		manifest := fmt.Sprintf(`{
  "plugins": {
    "fc": {
      "name": "fc",
      "version": "1.0.0",
      "path": %q
    }
  }
}`, pluginPath)
		if err := os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(manifest), 0644); err != nil {
			t.Fatalf("write manifest: %v", err)
		}

		ctx, command := newSafetyCommandoTestCtx(t)
		os.Args = []string{"aliyun", "fc", "create-function"}
		args := []string{"fc", "create-function"}

		err := command.main(ctx, args)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "blocked by safety policy")
		assert.Contains(t, err.Error(), "fc:create-function")
	})
}
