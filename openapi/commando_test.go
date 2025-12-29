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
	"errors"
	"os"
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

	EndpointFlag(ctx.Flags()).SetAssigned(true)
	EndpointFlag(ctx.Flags()).SetValue("ecs.cn-hangzhou.aliyuncs")

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
		EndpointFlag(ctx.Flags()).SetAssigned(true)
		EndpointFlag(ctx.Flags()).SetValue("ecs.cn-hangzhou.aliyuncs.com")
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
		assert.NoError(t, err) // Should succeed with force flag
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
		EndpointFlag(ctx.Flags()).SetAssigned(true)
		EndpointFlag(ctx.Flags()).SetValue("ecs.cn-hangzhou.aliyuncs.com")

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
		EndpointFlag(ctx.Flags()).SetAssigned(true)
		EndpointFlag(ctx.Flags()).SetValue("ecs.cn-hangzhou.aliyuncs.com")

		DryRunFlag(ctx.Flags()).SetAssigned(true)
		args := []string{"ecs", "GET", "/instances"}
		err := command.main(ctx, args)
		assert.NoError(t, err)
	})
}
