package openapi

import (
	"bytes"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/config"
	"github.com/aliyun/aliyun-cli/i18n"
	"github.com/aliyun/aliyun-cli/meta"
	"github.com/stretchr/testify/assert"

	"testing"
)

func Test_main(t *testing.T) {
	w := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w)
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
	profileflag.SetAssigned(true)
	profileflag.SetValue("ecs")
	skipflag := config.NewSkipSecureVerify()
	skipflag.SetAssigned(true)

	ctx.Flags().Add(profileflag)
	ctx.Flags().Add(skipflag)
	ctx.Flags().Add(config.NewRegionFlag())

	err = command.main(ctx, args)
	assert.NotNil(t, err)
	assert.Equal(t, "unknown profile ecs, run configure to check", err.Error())

	//args = []string{"test"}
	//profileflag.SetAssigned(false)
	//err = command.main(ctx, args)
	//assert.NotNil(t, err)
	//assert.Equal(t, "'test' is not a valid command or product. See `aliyun help`.", err.Error())
	//
	//args = []string{"aos", "test2"}
	//err = command.main(ctx, args)
	//assert.NotNil(t, err)
	//assert.Equal(t, "'aos' is not a valid command or product. See `aliyun help`.", err.Error())
	//
	//reader := &reader_test{}
	//reader.content = `{"products":[{"code":"aos","api_style":"restful"}]}`
	//command.library.builtinRepo = meta.LoadRepository(reader)
	//err = command.main(ctx, args)
	//assert.NotNil(t, err)
	//assert.Equal(t, "missing version for product aos", err.Error())
	//
	//args = []string{"test", "test2", "test1"}
	//err = command.main(ctx, args)
	//assert.NotNil(t, err)
	//assert.Equal(t, "'test' is not a valid command or product. See `aliyun help`.", err.Error())
	//
	//args = []string{"test", "test2", "test1", "test3"}
	//err = command.main(ctx, args)
	//assert.NotNil(t, err)
	//assert.Equal(t, "too many arguments", err.Error())
}

func Test_processInvoke(t *testing.T) {
	w := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w)
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

	HeaderFlag(ctx.Flags()).SetValues([]string{"Accept=xml","Content-Type=json", })

	profile := config.Profile{
		Language: "en",
		Mode: "AK",
		AccessKeyId: "accesskeyid",
		AccessKeySecret: "accesskeysecret",
		RegionId: "cn-hangzhou",
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
	defer func(){
		hookdo = originhookdo
	}()
	hookdo = func(fn func()(*responses.CommonResponse, error)) func()(*responses.CommonResponse, error) {
		resp := responses.NewCommonResponse()
		return  func()(*responses.CommonResponse, error){
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
	assert.Equal(t, "unmarshal output failed unexpected end of JSON input", err.Error())

	OutputFlag(ctx.Flags()).SetAssigned(false)
	err = command.processInvoke(ctx, productCode, apiOrMethod, path)
	assert.Nil(t, err)

	out := `{"requestid":"test","name":"json"}`
	out = FormatJson(out)
	assert.Equal(t, "{\n\t\"requestid\": \"test\",\n\t\"name\": \"json\"\n}", out)
}

func Test_help(t *testing.T){
	w := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w)
	cmd := &cli.Command{}
	cmd.EnableUnknownFlag = true

	AddFlags(cmd.Flags())
	ctx.EnterCommand(cmd)
	ctx.Command().Short = &i18n.Text{}

	profile := config.Profile{
		Language: "en",
		Mode: "AK",
		AccessKeyId: "accesskeyid",
		AccessKeySecret: "accesskeysecret",
		RegionId: "cn-hangzhou",
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

func Test_complete(t *testing.T){
	w := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w)
	cmd := &cli.Command{}
	cmd.EnableUnknownFlag = true

	AddFlags(cmd.Flags())
	ctx.EnterCommand(cmd)
	ctx.Command().Short = &i18n.Text{}
	ctx.SetCompletion(&cli.Completion{
		Current: "aos",
	})

	profile := config.Profile{
		Language: "en",
		Mode: "AK",
		AccessKeyId: "accesskeyid",
		AccessKeySecret: "accesskeysecret",
		RegionId: "cn-hangzhou",
	}
	command := NewCommando(w, profile)
	reader := &reader_test{}
	reader.content = `{"products":[{"code":"ecs","api_style":"rpc","apis":["DescribeRegions","Copy"]},{"code":"aos","api_style":"restful"}]}`
	command.library.builtinRepo = meta.LoadRepository(reader)
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