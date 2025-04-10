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
	"strings"
	"testing"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
	"github.com/stretchr/testify/assert"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/aliyun/aliyun-cli/v3/meta"
)

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
	assert.Equal(t, "'aos' is not a valid command or product. See `aliyun help`.", err.Error())

	args = []string{"test", "test2", "test1"}
	err = command.main(ctx, args)
	assert.NotNil(t, err)
	assert.Equal(t, "can not find api by path test1", err.Error())

	args = []string{"test", "test2", "test1", "test3"}
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

	EndpointFlag(ctx.Flags()).SetAssigned(true)
	EndpointFlag(ctx.Flags()).SetValue("ecs.cn-hangzhou.aliyuncs")

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
	assert.Equal(t, "'test' is not a valid command or product. See `aliyun help`.", err.Error())

	args = []string{"test", "test0"}
	err = command.help(ctx, args)
	assert.NotNil(t, err)
	assert.Equal(t, "'test' is not a valid command or product. See `aliyun help`.", err.Error())

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
