package openapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/cli/plugin"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/aliyun/aliyun-cli/v3/meta"
	"github.com/aliyun/aliyun-openapi-runtime/engine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandoConfigurationSettersCopyInputs(t *testing.T) {
	c, _, _ := newTestCommando()
	validator := func(RecoverySearchRequest) bool { return true }
	c.SetRecoverySearchValidator(validator)
	require.NotNil(t, c.recoverySearchValidator)
	assert.True(t, c.recoverySearchValidator(RecoverySearchRequest{}))

	commands := []RootCommandSpec{{Path: []string{"configure"}}}
	flags := []RootFlagSpec{{Name: "--profile"}}
	c.SetRootHelpSpecs(commands, flags)
	commands[0] = RootCommandSpec{Path: []string{"mutated"}}
	flags[0].Name = "--mutated"
	assert.Equal(t, "configure", c.rootCommandHelpSpecs[0].Path[0])
	assert.Equal(t, "--profile", c.rootFlagHelpSpecs[0].Name)
}

func TestLegacyHelpCanonicalRootProductRequestAndResponse(t *testing.T) {
	t.Run("root search", func(t *testing.T) {
		c, ctx, stdout, _ := newCanonicalHelpTestContext(t)
		CliHelpSearchFlag(ctx.Flags()).SetAssigned(true)
		CliHelpSearchFlag(ctx.Flags()).SetValue("demo")
		require.NoError(t, c.legacyHelp(ctx, nil))
		assert.Contains(t, stdout.String(), "demo")
		assert.Contains(t, stdout.String(), cli.AIModeEnableCommand)
	})

	t.Run("product all", func(t *testing.T) {
		c, ctx, stdout, _ := newCanonicalHelpTestContext(t)
		CliHelpAllFlag(ctx.Flags()).SetAssigned(true)
		require.NoError(t, c.legacyHelp(ctx, []string{"demo"}))
		assert.Contains(t, stdout.String(), "Product: demo")
		assert.Contains(t, stdout.String(), "DescribeRegions")
	})

	t.Run("request search", func(t *testing.T) {
		c, ctx, stdout, _ := newCanonicalHelpTestContext(t)
		c.library.helpRepo = canonicalmeta.NewRepository(os.DirFS("../aliyun-openapi-meta"))
		c.library.baselineHelpRepo = c.library.helpRepo
		c.library.builtinRepo = getRepository()
		CliHelpSearchFlag(ctx.Flags()).SetAssigned(true)
		CliHelpSearchFlag(ctx.Flags()).SetValue("instance id")
		require.NoError(t, c.legacyHelp(ctx, []string{"ecs", "DescribeInstances"}))
		assert.Contains(t, stdout.String(), "InstanceId")
	})

	t.Run("response search", func(t *testing.T) {
		c, ctx, stdout, _ := newCanonicalHelpTestContext(t)
		CliHelpSectionFlag(ctx.Flags()).SetAssigned(true)
		CliHelpSectionFlag(ctx.Flags()).SetValue("response")
		CliHelpSearchFlag(ctx.Flags()).SetAssigned(true)
		CliHelpSearchFlag(ctx.Flags()).SetValue("report-id")
		require.NoError(t, c.legacyHelp(ctx, []string{"demo", "CreateReport"}))
		assert.Contains(t, stdout.String(), "ReportId")
	})

	t.Run("machine JSON", func(t *testing.T) {
		c, ctx, stdout, _ := newCanonicalHelpTestContext(t)
		MachineHelpFormatFlag(ctx.Flags()).SetAssigned(true)
		MachineHelpFormatFlag(ctx.Flags()).SetValue("json")
		require.NoError(t, c.legacyHelp(ctx, []string{"demo"}))
		assert.Contains(t, stdout.String(), `"kind": "product"`)
	})
}

func TestCommandoPrintUsageAndNonInteractiveInstallHint(t *testing.T) {
	c, stdout, stderr := newTestCommando()
	command := testMachineHelpRootCommand()
	ctx := cli.NewCommandContext(stdout, stderr)
	ctx.EnterCommand(command)
	c.printUsage(ctx)
	assert.Contains(t, stdout.String(), "Alibaba Cloud CLI")
	assert.Contains(t, stdout.String(), "configure")

	pluginName, err := c.promptInstallInNonInteractiveMode(ctx, "aliyun-cli-demo", "demo list")
	require.NoError(t, err)
	assert.Empty(t, pluginName)
	assert.Contains(t, stderr.String(), "aliyun plugin install --names aliyun-cli-demo")
	assert.Contains(t, stderr.String(), "--enable-pre")
}

func TestMarshalDryRunInvokeMeta(t *testing.T) {
	request := requests.NewCommonRequest()
	request.Product = "ecs"
	request.Version = "2014-05-26"
	request.ApiName = "DescribeRegions"
	request.RegionId = "cn-hangzhou"
	request.Domain = "ecs.cn-hangzhou.aliyuncs.com"
	invoker := &RpcInvoker{BasicInvoker: &BasicInvoker{request: request}}

	encoded, err := marshalDryRunInvokeMeta(nil, invoker)
	require.NoError(t, err)
	var document dryRunInvokeMeta
	require.NoError(t, json.Unmarshal([]byte(encoded), &document))
	assert.Equal(t, "DescribeRegions", document.API)
	assert.Equal(t, "cn-hangzhou", document.Region)
}

func TestCheckApiParamWithBuiltInArgsCopiesOnlyQueryValues(t *testing.T) {
	c := &Commando{}
	ctx := cli.NewCommandContext(&bytes.Buffer{}, &bytes.Buffer{})
	ctx.SetUnknownFlags(cli.NewFlagSet())
	query := &cli.Flag{Name: "RegionId"}
	query.SetAssigned(true)
	query.SetValue("cn-hangzhou")
	body := &cli.Flag{Name: "Payload"}
	body.SetAssigned(true)
	body.SetValue("body")
	ctx.Flags().Add(query)
	ctx.Flags().Add(body)
	api := &canonicalAPIForCoverage
	c.CheckApiParamWithBuildInArgs(ctx, api.value())
	assert.Equal(t, "cn-hangzhou", assignedFlagValue(ctx.UnknownFlags(), "RegionId"))
	assert.Empty(t, assignedFlagValue(ctx.UnknownFlags(), "Payload"))
	c.CheckApiParamWithBuildInArgs(ctx, nil)
}

type canonicalAPIForCoverageType struct{}

var canonicalAPIForCoverage canonicalAPIForCoverageType

func (canonicalAPIForCoverageType) value() *canonicalmeta.API {
	return &canonicalmeta.API{Parameters: []canonicalmeta.Parameter{
		{RawName: "RegionId", Type: "string", Location: "query"},
		{RawName: "Payload", Type: "string", Location: "body"},
	}}
}

func assignedFlagValue(flags *cli.FlagSet, name string) string {
	flag := flags.Get(name)
	if flag == nil || !flag.IsAssigned() {
		return ""
	}
	value, _ := flag.GetValue()
	return strings.TrimSpace(value)
}

func TestLegacyHelpFallbackRootProductAPIAndTooManyArguments(t *testing.T) {
	repo, err := meta.MockLoadRepository([]meta.Product{{
		Code: "demo", Version: "2026-01-01", ApiStyle: "rpc",
		ApiNames: []string{"CreateReport"}, Name: map[string]string{"en": "Demo Service"},
	}})
	require.NoError(t, err)
	canonical := newFakeCanonicalRepo()
	canonical.AddAPI("demo", "2026-01-01", &canonicalmeta.API{
		Name: "CreateReport", Protocol: "HTTPS", Method: "POST",
		Parameters: []canonicalmeta.Parameter{{Name: "ReportId", RawName: "ReportId", Type: "string", Location: "query"}},
	})

	for _, tc := range []struct {
		name     string
		args     []string
		contains string
	}{
		{name: "root", contains: "Products:"},
		{name: "product", args: []string{"demo"}, contains: "Available Api List:"},
		{name: "api", args: []string{"demo", "CreateReport"}, contains: "--ReportId"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, ctx, stdout, _ := newCanonicalHelpTestContext(t)
			c.library.builtinRepo = repo
			c.library.canonicalRepo = canonical
			c.pluginLoaded = true
			require.NoError(t, c.legacyHelp(ctx, tc.args))
			assert.Contains(t, stdout.String(), tc.contains)
			assert.Contains(t, stdout.String(), cli.AIModeEnableCommand)
		})
	}

	c, ctx, _, _ := newCanonicalHelpTestContext(t)
	c.library.builtinRepo = repo
	c.pluginLoaded = true
	err = c.legacyHelp(ctx, []string{"demo", "CreateReport", "nested"})
	assert.ErrorContains(t, err, "too many arguments: 3")
}

func TestLegacyHelpAIModeDefaultCanonicalViews(t *testing.T) {
	repo, err := meta.MockLoadRepository([]meta.Product{{
		Code: "demo", Version: "2026-01-01", ApiStyle: "rpc",
		ApiNames: []string{"CreateReport", "DescribeRegions"},
	}})
	require.NoError(t, err)
	canonical := canonicalmeta.NewRepository(os.DirFS("../canonicalmeta/testdata"))

	for _, tc := range []struct {
		name     string
		args     []string
		contains string
	}{
		{name: "root", contains: "demo"},
		{name: "product", args: []string{"demo"}, contains: "Product: demo"},
		{name: "api", args: []string{"demo", "CreateReport"}, contains: "--ReportId"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, ctx, stdout, _ := newCanonicalHelpTestContext(t)
			t.Setenv("ALIBABA_CLOUD_CLI_AI_MODE", "1")
			c.library.builtinRepo = repo
			c.library.canonicalRepo = canonical
			c.pluginLoaded = true
			require.NoError(t, c.legacyHelp(ctx, tc.args))
			assert.Contains(t, stdout.String(), tc.contains)
			assert.NotContains(t, stdout.String(), cli.AIModeEnableCommand)
		})
	}
}

func TestCommandoCompletionCoversProductsParametersAndRestfulMethods(t *testing.T) {
	products, err := meta.MockLoadRepository([]meta.Product{
		{Code: "ecs", Version: "2014-05-26", ApiStyle: "rpc", ApiNames: []string{"DescribeInstances"}},
		{Code: "cs", Version: "2015-12-15", ApiStyle: "restful", ApiNames: []string{"DescribeCluster"}},
	})
	require.NoError(t, err)
	canonical := newFakeCanonicalRepo()
	canonical.AddAPI("ecs", "2014-05-26", &canonicalmeta.API{
		Name: "DescribeInstances",
		Parameters: []canonicalmeta.Parameter{
			{Name: "InstanceId", RawName: "InstanceId", Type: "string", Location: "query"},
			{Name: "Host", RawName: "Host", Type: "string", Location: "header"},
			{Name: "Domain", RawName: "Domain", Type: "string", Location: "domain"},
		},
	})
	c := &Commando{library: &Library{builtinRepo: products, canonicalRepo: canonical}}
	stdout := &bytes.Buffer{}
	ctx := cli.NewCommandContext(stdout, &bytes.Buffer{})
	root := testMachineHelpRootCommand()
	root.AddSubCommand(&cli.Command{Name: "configure"})
	ctx.EnterCommand(root)

	ctx.SetCompletion(&cli.Completion{Current: "ec"})
	assert.Empty(t, c.complete(ctx, nil))
	assert.Contains(t, stdout.String(), "ecs")
	stdout.Reset()

	ctx.SetCompletion(&cli.Completion{Current: "--In"})
	assert.Empty(t, c.complete(ctx, []string{"ecs", "DescribeInstances"}))
	assert.Equal(t, "--InstanceId\n", stdout.String())
	stdout.Reset()

	ctx.SetCompletion(&cli.Completion{})
	assert.Empty(t, c.complete(ctx, []string{"cs"}))
	assert.Contains(t, stdout.String(), "GET\n")
	assert.Contains(t, stdout.String(), "POST\n")
	assert.Empty(t, c.complete(ctx, []string{"missing"}))
}

func TestCommandoSmallRoutingHelpers(t *testing.T) {
	c := &Commando{rootCommandHelpSpecs: []RootCommandSpec{
		{Group: RootGroupExtension, Path: []string{"plugin"}},
		{Group: RootGroupCore, Path: []string{"configure"}},
	}}
	assert.False(t, c.isExtensionInvocation(nil))
	assert.True(t, c.isExtensionInvocation([]string{"plugin", "list"}))
	assert.False(t, c.isExtensionInvocation([]string{"configure"}))

	cause := errors.New("ordinary")
	assert.NoError(t, c.adaptEngineUnknownCommand(nil))
	assert.Same(t, cause, c.adaptEngineUnknownCommand(cause))
	adapted := c.adaptEngineUnknownCommand(&engine.UnknownCommandError{Product: "ecs", Command: "describe-instnaces"})
	var invalid *InvalidBaselineCommandError
	require.ErrorAs(t, adapted, &invalid)
	assert.Equal(t, "ecs", invalid.Product)
	assert.Equal(t, "describe-instnaces", invalid.Command)

	assert.Equal(t, "", buildCommandName(nil))
	assert.Equal(t, "ecs", buildCommandName([]string{"ecs"}))
	assert.Equal(t, "ecs DescribeInstances", buildCommandName([]string{"ecs", "DescribeInstances", "ignored"}))
}

func TestRecoveryNormalizationArgsAndDryRunPathLookup(t *testing.T) {
	ctx := cli.NewCommandContext(&bytes.Buffer{}, &bytes.Buffer{})
	root := testMachineHelpRootCommand()
	AddFlags(root.Flags())
	ctx.EnterCommand(root)

	assert.Equal(t, []string{"ecs", "DescribeInstances"}, recoveryNormalizationArgs(ctx,
		[]string{"--cli-ai-mode", "ecs", "DescribeInstances", "--no-cli-ai-mode"}))
	VersionFlag(ctx.Flags()).SetAssigned(true)
	VersionFlag(ctx.Flags()).SetValue("2014-05-26")
	assert.Equal(t, []string{"ecs", "DescribeInstances", "--version", "2014-05-26"},
		recoveryNormalizationArgs(ctx, []string{"ecs", "DescribeInstances"}))
	assert.Equal(t, []string{"ecs", "describe-instances", "--api-version", "2014-05-26"},
		recoveryNormalizationArgs(ctx, []string{"ecs", "describe-instances"}))
	assert.Equal(t, []string{"ecs", "DescribeInstances", "--version", "2020-01-01"},
		recoveryNormalizationArgs(ctx, []string{"ecs", "DescribeInstances", "--version", "2020-01-01"}))
	CliHelpSectionFlag(ctx.Flags()).SetAssigned(true)
	CliHelpSectionFlag(ctx.Flags()).SetValue("response")
	CliOutputFlag(ctx.Flags()).SetAssigned(true)
	CliOutputFlag(ctx.Flags()).SetValue("json")
	assert.Equal(t,
		[]string{"help", "ecs", "DescribeInstances", "--version", "2014-05-26", "--cli-section", "response", "--cli-output", "json"},
		recoveryNormalizationArgs(ctx, []string{"help", "ecs", "DescribeInstances"}))
	assert.Equal(t,
		[]string{"help", "ecs", "DescribeInstances", "--version", "2014-05-26", "--cli-section", "response"},
		recoveryNormalizationArgs(ctx, []string{"help", "ecs", "DescribeInstances", "--cli-output=json"}, true))
	assert.Equal(t, []string{"ecs"}, recoveryNormalizationArgs(nil, []string{"ecs"}))

	repo, err := meta.MockLoadRepository([]meta.Product{{
		Code: "cs", Version: "2015-12-15", ApiStyle: "restful", ApiNames: []string{"DescribeCluster"},
	}})
	require.NoError(t, err)
	canonical := newFakeCanonicalRepo()
	canonical.AddAPI("cs", "2015-12-15", &canonicalmeta.API{
		Name: "DescribeCluster", Method: "GET", PathPattern: "/clusters/[ClusterId]",
	})
	library := &Library{builtinRepo: repo, canonicalRepo: canonical}
	assert.Equal(t, "DescribeCluster", dryRunRestfulAPIByPath(library, "cs", "2015-12-15", "GET", "/clusters/c-1"))
	assert.Equal(t, "GET /missing", dryRunRestfulAPIByPath(library, "cs", "2015-12-15", "GET", "/missing"))
	assert.Equal(t, "GET /clusters", dryRunRestfulAPIByPath(nil, "cs", "2015-12-15", "GET", "/clusters"))
	assert.Equal(t, "/clusters", dryRunRestfulAPIByPath(library, "cs", "2015-12-15", "", "/clusters"))
}

func TestPrintProductsCoversBuiltinAndPluginCombinations(t *testing.T) {
	previousLanguage := i18n.GetLanguage()
	t.Cleanup(func() { i18n.SetLanguage(previousLanguage) })
	i18n.SetLanguage("en")

	builtin, err := meta.MockLoadRepository([]meta.Product{
		{Code: "both-installed", Name: map[string]string{"en": "Built-in installed"}},
		{Code: "both-missing", Name: map[string]string{"en": "Built-in missing"}},
		{Code: "builtin-only", Name: map[string]string{"en": "Built-in only"}},
	})
	require.NoError(t, err)
	c := &Commando{
		library: &Library{builtinRepo: builtin},
		pluginIndex: &plugin.Index{Plugins: []plugin.PluginInfo{
			{Name: "aliyun-cli-both-installed", ProductCode: "both-installed", ProductName: map[string]string{"en": "Localized installed"}},
			{Name: "aliyun-cli-both-missing", ProductCode: "both-missing", Description: "Available plugin"},
			{Name: "aliyun-cli-plugin-missing", ProductCode: "plugin-missing"},
			{Name: "aliyun-cli-plugin-installed", ProductCode: "plugin-installed", Description: "Installed plugin"},
		}},
		localManifest: &plugin.LocalManifest{Plugins: map[string]plugin.LocalPlugin{
			"aliyun-cli-both-installed":   {Name: "aliyun-cli-both-installed"},
			"aliyun-cli-plugin-installed": {Name: "aliyun-cli-plugin-installed"},
		}},
	}
	stdout := &bytes.Buffer{}
	ctx := cli.NewCommandContext(stdout, &bytes.Buffer{})
	c.printProducts(ctx)

	output := stdout.String()
	assert.Contains(t, output, "Localized installed (Plugin: aliyun-cli-both-installed)")
	assert.Contains(t, output, "Available plugin (Plugin available but not installed: aliyun-cli-both-missing)")
	assert.Contains(t, output, "plugin-missing (Plugin: aliyun-cli-plugin-missing, Not Installed)")
	assert.Contains(t, output, "Installed plugin (Plugin: aliyun-cli-plugin-installed)")
	assert.Contains(t, output, "Built-in only")
	assert.Contains(t, output, "aliyun plugin install --names <plugin_name>")
}

func TestPreviouslyUncalledOpenAPICompatibilityHelpers(t *testing.T) {
	flag := NewMachineHelpFormatFlag()
	assert.Equal(t, CliOutputFlagName, flag.Name)

	products, err := meta.MockLoadRepository([]meta.Product{{
		Code: "cs", Version: "v1", ApiStyle: "restful", ApiNames: []string{"GetThing"},
	}})
	require.NoError(t, err)
	repository := newFakeCanonicalRepo()
	repository.AddAPI("cs", "v1", &canonicalmeta.API{Name: "GetThing", Method: "GET", PathPattern: "/things/[Id]"})
	library := &Library{builtinRepo: products, canonicalRepo: repository}
	assert.Equal(t, "GetThing", library.GetCanonicalApiByPath("cs", "v1", "GET", "/things/1").Name)
	assert.Nil(t, library.GetCanonicalApiByPath("cs", "v1", "GET", "/missing"))
}

func TestCreateInvokerAdditionalRoutingAndErrorBranches(t *testing.T) {
	products, err := meta.MockLoadRepository([]meta.Product{
		{Code: "ecs", Version: "2014-05-26", ApiStyle: "rpc", ApiNames: []string{"DescribeInstances"}},
		{Code: "cs", Version: "2015-12-15", ApiStyle: "restful", ApiNames: []string{"DescribeCluster"}},
	})
	require.NoError(t, err)
	profile := config.Profile{
		Mode: config.AK, AccessKeyId: "test-ak", AccessKeySecret: "test-sk", RegionId: "cn-hangzhou", Endpoint: "service.aliyuncs.com",
	}
	newCase := func() (*Commando, *cli.Context) {
		ctx := cli.NewCommandContext(&bytes.Buffer{}, &bytes.Buffer{})
		root := &cli.Command{Name: "DescribeCluster", EnableUnknownFlag: true}
		config.AddFlags(root.Flags())
		AddFlags(root.Flags())
		root.Flags().Add(&cli.Flag{Name: "style"})
		ctx.EnterCommand(root)
		return &Commando{
			profile: profile, pluginLoaded: true,
			library: &Library{builtinRepo: products, canonicalRepo: newFakeCanonicalRepo()},
		}, ctx
	}

	t.Run("unknown product without force", func(t *testing.T) {
		c, ctx := newCase()
		_, err := c.createInvoker(ctx, "missing", "DescribeThings", "")
		var invalid *InvalidProductError
		require.ErrorAs(t, err, &invalid)
	})

	t.Run("unknown rpc api without installed plugin", func(t *testing.T) {
		c, ctx := newCase()
		_, err := c.createInvoker(ctx, "ecs", "MissingAPI", "")
		var invalid *InvalidApiError
		require.ErrorAs(t, err, &invalid)
	})

	t.Run("unknown rpc api with installed plugin", func(t *testing.T) {
		c, ctx := newCase()
		c.localManifest = &plugin.LocalManifest{Plugins: map[string]plugin.LocalPlugin{
			"aliyun-cli-ecs": {Name: "aliyun-cli-ecs", CmdNames: []string{"describe-instances"}},
		}}
		_, err := c.createInvoker(ctx, "ecs", "MissingAPI", "")
		var invalid *InvalidUnifiedApiError
		require.ErrorAs(t, err, &invalid)
	})

	t.Run("force unknown version requires style", func(t *testing.T) {
		c, ctx := newCase()
		ForceFlag(ctx.Flags()).SetAssigned(true)
		VersionFlag(ctx.Flags()).SetAssigned(true)
		VersionFlag(ctx.Flags()).SetValue("2099-01-01")
		_, err := c.createInvoker(ctx, "ecs", "DescribeThings", "")
		assert.ErrorContains(t, err, "uncheked version")

		ctx.Flags().Get("style").SetAssigned(true)
		ctx.Flags().Get("style").SetValue("rpc")
		invoker, err := c.createInvoker(ctx, "ecs", "DescribeThings", "")
		require.NoError(t, err)
		assert.IsType(t, &ForceRpcInvoker{}, invoker)
	})

	t.Run("restful method required and accepted", func(t *testing.T) {
		c, ctx := newCase()
		_, err := c.createInvoker(ctx, "cs", "not-a-method", "")
		assert.ErrorContains(t, err, "need restful call")
		invoker, err := c.createInvoker(ctx, "cs", "GET", "/clusters")
		require.NoError(t, err)
		assert.IsType(t, &RestfulInvoker{}, invoker)
	})

	t.Run("forced unknown product supports restful and rpc", func(t *testing.T) {
		c, ctx := newCase()
		ForceFlag(ctx.Flags()).SetAssigned(true)
		VersionFlag(ctx.Flags()).SetAssigned(true)
		VersionFlag(ctx.Flags()).SetValue("2026-01-01")
		invoker, err := c.createInvoker(ctx, "custom", "GET", "/things")
		require.NoError(t, err)
		assert.IsType(t, &RestfulInvoker{}, invoker)
		invoker, err = c.createInvoker(ctx, "custom", "DoThing", "")
		require.NoError(t, err)
		assert.IsType(t, &ForceRpcInvoker{}, invoker)
	})
}
