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
package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/cli/plugin"
	"github.com/aliyun/aliyun-cli/v3/cli/upgrade"
	"github.com/aliyun/aliyun-cli/v3/cliext"
	"github.com/aliyun/aliyun-cli/v3/cliext/acrutil"
	"github.com/aliyun/aliyun-cli/v3/cliext/agentbay"
	"github.com/aliyun/aliyun-cli/v3/cliext/appmanagerutil"
	"github.com/aliyun/aliyun-cli/v3/cliext/cms2"
	"github.com/aliyun/aliyun-cli/v3/cliext/codeup"
	"github.com/aliyun/aliyun-cli/v3/cliext/computenestutil"
	"github.com/aliyun/aliyun-cli/v3/cliext/ecctl"
	"github.com/aliyun/aliyun-cli/v3/cliext/esacli"
	"github.com/aliyun/aliyun-cli/v3/cliext/flowcli"
	"github.com/aliyun/aliyun-cli/v3/cliext/iact3"
	"github.com/aliyun/aliyun-cli/v3/cliext/kmscli"
	"github.com/aliyun/aliyun-cli/v3/cliext/lindormcli"
	"github.com/aliyun/aliyun-cli/v3/cliext/maxc"
	"github.com/aliyun/aliyun-cli/v3/cliext/mseutil"
	"github.com/aliyun/aliyun-cli/v3/cliext/ossutil"
	"github.com/aliyun/aliyun-cli/v3/cliext/otsutil"
	"github.com/aliyun/aliyun-cli/v3/cliext/rostran"
	"github.com/aliyun/aliyun-cli/v3/cliext/saectl"
	"github.com/aliyun/aliyun-cli/v3/cliext/sparksubmit"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/export"
	go_migrate "github.com/aliyun/aliyun-cli/v3/go-migrate"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/aliyun/aliyun-cli/v3/mcpproxy"
	"github.com/aliyun/aliyun-cli/v3/mock"
	"github.com/aliyun/aliyun-cli/v3/openapi"
	"github.com/aliyun/aliyun-cli/v3/oss/lib"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/aimode"
	sysmock "github.com/aliyun/aliyun-cli/v3/sysconfig/mock"
	"github.com/aliyun/aliyun-cli/v3/util"
)

var (
	newStdoutWriter = cli.DefaultStdoutWriter
	newStderrWriter = cli.DefaultStderrWriter
	exit            = cli.Exit
)

func Main(args []string) {
	stdout := newStdoutWriter()
	stderr := newStderrWriter()

	if sysmock.FirstCommandToken(args) != "mock" {
		result := sysmock.Intercept(sysmock.Options{
			Args:     args,
			Stdout:   stdout,
			Stderr:   stderr,
			MockPath: sysmock.ResolvePath(config.GetConfigPath),
		})
		if result.Handled {
			exit(result.ExitCode)
			return
		}
	}

	// load current configuration
	profile, err := config.LoadOrCreateDefaultProfile()
	if err != nil {
		if startupAIMode(args) {
			// Configuration errors can contain configuration values. Keep the
			// machine diagnostic stable without echoing potentially secret data.
			agentErr := cli.NewAgentError(cli.AgentErrorEnvelope{
				Message:   "load current configuration failed",
				ErrorCode: "ConfigurationError",
				Recovery: cli.AgentErrorRecovery{
					Action: "check_configuration",
					Hint:   "Check that the CLI configuration is readable, contains valid JSON, and selects an existing profile; repair it before retrying.",
				},
			}, err).WithSchemaVersion()
			_ = json.NewEncoder(stderr).Encode(agentErr.Envelope())
		} else {
			cli.Errorf(stderr, "ERROR: load current configuration failed %s", err)
		}
		exit(1)
		return
	}

	// Resolve language before command routing so Core and OpenAPI Help observe
	// the same deterministic priority: explicit flag > profile > system locale.
	i18n.SetLanguage(effectiveLanguage(args, profile.Language))

	rootCmd := newRootCommand(profile, stdout)

	ctx := newCommandContext(stdout, stderr)
	ctx.EnterCommand(rootCmd)
	ctx.SetCompletion(cli.ParseCompletionForShell())
	ctx.SetInConfigureMode(openapi.DetectInConfigureMode(ctx.Flags()))
	// use http force, current use in oss bridge
	insecure, _ := ParseInSecure(args)
	ctx.SetInsecure(insecure)

	if os.Getenv("GENERATE_METADATA") == "YES" {
		generateMetadata(rootCmd)
	} else {
		rootCmd.Execute(ctx, args)
	}
}

func effectiveLanguage(args []string, profileLanguage string) string {
	for i, arg := range args {
		value := ""
		switch {
		case strings.HasPrefix(arg, "--language="):
			value = strings.TrimSpace(strings.TrimPrefix(arg, "--language="))
		case arg == "--language" && i+1 < len(args):
			value = strings.TrimSpace(args[i+1])
		}
		if value == "" {
			continue
		}
		switch strings.ToLower(value) {
		case string(i18n.Zh):
			return string(i18n.Zh)
		case string(i18n.En):
			return string(i18n.En)
		default:
			// Keep the existing CLI behavior for unsupported language values:
			// fall back to English instead of allowing the profile to win.
			return string(i18n.En)
		}
	}
	return profileLanguage
}

func newCommandContext(stdout io.Writer, stderr io.Writer) *cli.Context {
	ctx := cli.NewCommandContext(stdout, stderr)
	ctx.SetAgentName(util.DetectAgentName())
	return ctx
}

func newRootCommand(profile config.Profile, stdout io.Writer) *cli.Command {
	// create root command
	rootCmd := &cli.Command{
		Name:              "aliyun",
		Short:             i18n.T("Alibaba Cloud Command Line Interface Version "+cli.Version, "阿里云CLI命令行工具 "+cli.Version),
		Usage:             "aliyun <product> <operation> [--parameter1 value1 --parameter2 value2 ...]",
		Sample:            "aliyun ecs DescribeRegions",
		EnableUnknownFlag: true,
	}

	// add default flags
	config.AddFlags(rootCmd.Flags())
	openapi.AddFlags(rootCmd.Flags())

	// new open api commando to process rootCmd
	commando := openapi.NewCommando(stdout, profile)
	commando.InitWithCommand(rootCmd)

	rootCmd.AddSubCommand(config.NewConfigureCommand())
	utilsCmd, utilityAliases := newUtilsCommands()
	rootCmd.AddSubCommand(utilsCmd)
	for _, alias := range utilityAliases {
		rootCmd.AddSubCommand(alias)
	}
	// oss old version, duplicate with ossutil, will remove in future
	ossCmd := lib.NewOssCommand()
	// `aliyun oss <ApiName> ... --estimate-cost` quotes via CloudControl; the
	// bridge shadows the generic product routing, so wire the quote fallthrough
	// here (file-operation subcommands are matched first and stay untouched).
	commando.AttachOssEstimateCost(ossCmd)
	rootCmd.AddSubCommand(ossCmd)
	versionCmd := cli.NewVersionCommand()
	versionCmd.Hidden = false
	rootCmd.AddSubCommand(versionCmd)
	rootCmd.AddSubCommand(cli.NewAutoCompleteCommand())
	extensions := make(map[string]bool)
	addExtension := func(command *cli.Command) {
		extensions[command.Name] = true
		rootCmd.AddSubCommand(command)
	}
	// new oss command
	addExtension(ossutil.NewOssutilCommand())
	// AgentBay command
	addExtension(agentbay.NewAgentBayCommand())
	// tablestore command
	addExtension(otsutil.NewOtsutilCommand())
	// EMR Serverless spark-submit command
	addExtension(sparksubmit.NewSparkSubmitCommand())
	// kmscli command
	addExtension(kmscli.NewKmscliCommand())
	// lindorm command
	addExtension(lindormcli.NewLindormCliCommand())
	// mseutil command
	addExtension(mseutil.NewMseutilCommand())
	// acr command
	addExtension(acrutil.NewAcrutilCommand())
	// codeup command
	addExtension(codeup.NewCodeupCliCommand())
	// sae command
	addExtension(saectl.NewSaectlCommand())
	// appmanager command
	addExtension(appmanagerutil.NewAppManagerCommand())
	// computenest command
	addExtension(computenestutil.NewComputenestCommand())
	// ecctl command
	addExtension(ecctl.NewEcctlCommand())
	// esa-cli command
	addExtension(esacli.NewEsacliCommand())
	// flow-cli command (云效 Flow)
	addExtension(flowcli.NewFlowcliCommand())
	// cms2 command
	addExtension(cms2.NewCms2Command())
	// maxc command
	addExtension(maxc.NewMaxcCommand())
	// iact3 command
	addExtension(iact3.NewIact3Command())
	// rostran command
	addExtension(rostran.NewRostranCommand())
	// plugin command
	rootCmd.AddSubCommand(plugin.NewPluginCommand())
	// upgrade command
	rootCmd.AddSubCommand(upgrade.NewUpgradeCommand())
	// mock command
	rootCmd.AddSubCommand(mock.NewMockCommand(config.GetConfigPath))
	commando.SetRootHelpSpecs(rootCommandHelpSpecs, rootFlagHelpSpecs)

	cliext.AttachSafetyPolicy(rootCmd, extensions)

	plugin.RegisterReservedTopLevelCommands(rootCmd.SubCommandNames())

	return rootCmd
}

func ParseInSecure(args []string) (bool, interface{}) {
	// check has insecure flag
	for _, arg := range args {
		if arg == "--insecure" {
			return true, nil
		}
	}
	return false, nil
}

func main() {
	Main(os.Args[1:])
}

func dumpFiles(fs embed.FS, filePath string, outputDir string) {
	filePath = strings.TrimPrefix(filePath, "./")

	entries, err := fs.ReadDir(filePath)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	for _, entry := range entries {
		entryPath := path.Join(filePath, entry.Name())
		if entry.IsDir() {
			dumpFiles(fs, entryPath, outputDir)
		} else {
			content, err := fs.ReadFile(entryPath)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			targetPath := path.Join(outputDir, entryPath)
			fmt.Println("copy file from " + entryPath + " to " + targetPath)
			_, err = os.Stat(path.Dir(targetPath))
			if os.IsNotExist(err) {
				err = os.MkdirAll(path.Dir(targetPath), 0755)
				if err != nil {
					fmt.Println(err.Error())
					return
				}
			}
			err = os.WriteFile(targetPath, content, 0666)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
		}
	}
}

func generateMetadata(rootCmd *cli.Command) {
	metadata := make(map[string]*cli.Metadata)
	rootCmd.GetMetadata(metadata)
	b, _ := json.MarshalIndent(metadata, "", "  ")
	cwd, _ := os.Getwd()
	targetDir := cwd + "/cli-metadata"
	_, err := os.Stat(targetDir)
	if os.IsNotExist(err) {
		err := os.Mkdir(targetDir, 0755)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
	}

	targetPath := targetDir + "/commands.json"
	err = os.WriteFile(targetPath, b, 0666)
	if err != nil {
		fmt.Println(err.Error())
	}

	versionPath := targetDir + "/version"
	os.WriteFile(versionPath, []byte(cli.Version), 0666)

	if err := export.LegacyExportMetadata(targetDir); err != nil {
		fmt.Println(err.Error())
	}
}

// Keep the entrypoint self-contained for distributors building main/main.go directly.

type utilityCommandFactory func() *cli.Command

var utilityCommandFactories = []utilityCommandFactory{
	openapi.NewListSupportedPricingApisCommand,
	mcpproxy.NewMCPProxyCommand,
	go_migrate.NewGoMigrateCommand,
}

// newUtilsCommands creates distinct command instances for the canonical
// subtree and the hidden root compatibility entrypoints. Each pair comes from
// the same factory/handler implementation; sharing a *cli.Command would be
// unsafe because AddSubCommand assigns its parent and flags retain parse state.
func newUtilsCommands() (*cli.Command, []*cli.Command) {
	utils := &cli.Command{
		Name:  "utils",
		Short: i18n.T("Local Alibaba Cloud CLI utilities", "阿里云 CLI 本地工具"),
		Usage: "utils <name> [flags]",
	}
	aliases := make([]*cli.Command, 0, len(utilityCommandFactories))
	for _, factory := range utilityCommandFactories {
		canonical := factory()
		prepareUtilityCommand(canonical, true)
		utils.AddSubCommand(canonical)

		legacy := factory()
		prepareUtilityCommand(legacy, false)
		legacy.Hidden = true
		aliases = append(aliases, legacy)
	}
	return utils, aliases
}

func prepareUtilityCommand(command *cli.Command, canonical bool) {
	if command == nil {
		return
	}
	legacyPrefix := "aliyun " + command.Name
	if strings.HasPrefix(command.Usage, legacyPrefix) {
		command.Usage = command.Name + strings.TrimPrefix(command.Usage, legacyPrefix)
	}
	if canonical && command.Sample != "" {
		command.Sample = strings.ReplaceAll(command.Sample, legacyPrefix, "aliyun utils "+command.Name)
	}
}

var rootCommandHelpSpecs = []openapi.RootCommandSpec{
	{Path: []string{"configure"}, Group: openapi.RootGroupCore},
	{Path: []string{"plugin"}, Group: openapi.RootGroupCore},
	{Path: []string{"upgrade"}, Group: openapi.RootGroupCore},
	{Path: []string{"version"}, Group: openapi.RootGroupCore},
	{Path: []string{"auto-completion"}, Group: openapi.RootGroupCore},
	{Path: []string{"mock"}, Group: openapi.RootGroupCore},
	{Path: []string{"utils"}, Group: openapi.RootGroupCore},

	{Path: []string{"utils", "mcp-proxy"}, Group: openapi.RootGroupUtils, Aliases: []string{"mcp-proxy"}},
	{Path: []string{"utils", "go-migrate"}, Group: openapi.RootGroupUtils, Aliases: []string{"go-migrate"}},

	{Path: []string{"oss"}, Group: openapi.RootGroupExtension},
	{Path: []string{"ossutil"}, Group: openapi.RootGroupExtension},
	{Path: []string{"agentbay"}, Group: openapi.RootGroupExtension},
	{Path: []string{"otsutil"}, Group: openapi.RootGroupExtension},
	{Path: []string{"kmscli"}, Group: openapi.RootGroupExtension},
	{Path: []string{"lindorm"}, Group: openapi.RootGroupExtension},
	{Path: []string{"mseutil"}, Group: openapi.RootGroupExtension},
	{Path: []string{"acrutil"}, Group: openapi.RootGroupExtension},
	{Path: []string{"codeup-cli"}, Group: openapi.RootGroupExtension},
	{Path: []string{"saectl"}, Group: openapi.RootGroupExtension},
	{Path: []string{"appmanager"}, Group: openapi.RootGroupExtension},
	{Path: []string{"computenest-cli"}, Group: openapi.RootGroupExtension},
	{Path: []string{"ecctl"}, Group: openapi.RootGroupExtension},
	{Path: []string{"esa-cli"}, Group: openapi.RootGroupExtension},
	{Path: []string{"flow-cli"}, Group: openapi.RootGroupExtension},
	{Path: []string{"cms2"}, Group: openapi.RootGroupExtension},
	{Path: []string{"maxc"}, Group: openapi.RootGroupExtension},
	{Path: []string{"iact3"}, Group: openapi.RootGroupExtension},
	{Path: []string{"rostran"}, Group: openapi.RootGroupExtension},
}

var rootFlagHelpSpecs = []openapi.RootFlagSpec{
	{Name: "profile", Visibility: openapi.RootVisibilityDefault},
	{Name: "region", Visibility: openapi.RootVisibilityDefault},
	{Name: "language", Visibility: openapi.RootVisibilityDefault},
	{Name: "version", Visibility: openapi.RootVisibilityDefault},
	{Name: "output", Visibility: openapi.RootVisibilityDefault},
	{Name: "cli-query", Visibility: openapi.RootVisibilityDefault},
	{Name: "cli-output", Visibility: openapi.RootVisibilityDefault},
	{Name: "cli-dry-run", Visibility: openapi.RootVisibilityDefault},
	{Name: "yes", Visibility: openapi.RootVisibilityDefault},
	{Name: "cli-ai-mode", Visibility: openapi.RootVisibilityDefault},
	{Name: "help", Visibility: openapi.RootVisibilityDefault},
	{Name: "help-all", Visibility: openapi.RootVisibilityDefault},
	{Name: "help-search", Visibility: openapi.RootVisibilityDefault},

	{Name: "mode", Visibility: openapi.RootVisibilityExtended},
	{Name: "config-path", Visibility: openapi.RootVisibilityExtended},
	{Name: "access-key-id", Visibility: openapi.RootVisibilityExtended},
	{Name: "access-key-secret", Visibility: openapi.RootVisibilityExtended},
	{Name: "sts-token", Visibility: openapi.RootVisibilityExtended},
	{Name: "sts-region", Visibility: openapi.RootVisibilityExtended},
	{Name: "sts-endpoint", Visibility: openapi.RootVisibilityExtended},
	{Name: "ram-role-name", Visibility: openapi.RootVisibilityExtended},
	{Name: "ram-role-arn", Visibility: openapi.RootVisibilityExtended},
	{Name: "source-profile", Visibility: openapi.RootVisibilityExtended},
	{Name: "role-session-name", Visibility: openapi.RootVisibilityExtended},
	{Name: "external-id", Visibility: openapi.RootVisibilityExtended},
	{Name: "private-key", Visibility: openapi.RootVisibilityExtended},
	{Name: "key-pair-name", Visibility: openapi.RootVisibilityExtended},
	{Name: "read-timeout", Visibility: openapi.RootVisibilityExtended},
	{Name: "connect-timeout", Visibility: openapi.RootVisibilityExtended},
	{Name: "retry-count", Visibility: openapi.RootVisibilityExtended},
	{Name: "skip-secure-verify", Visibility: openapi.RootVisibilityExtended},
	{Name: "expired-seconds", Visibility: openapi.RootVisibilityExtended},
	{Name: "process-command", Visibility: openapi.RootVisibilityExtended},
	{Name: "oidc-provider-arn", Visibility: openapi.RootVisibilityExtended},
	{Name: "oidc-token-file", Visibility: openapi.RootVisibilityExtended},
	{Name: "cloud-sso-sign-in-url", Visibility: openapi.RootVisibilityExtended},
	{Name: "cloud-sso-access-config", Visibility: openapi.RootVisibilityExtended},
	{Name: "cloud-sso-account-id", Visibility: openapi.RootVisibilityExtended},
	{Name: "oauth-site-type", Visibility: openapi.RootVisibilityExtended},
	{Name: "endpoint-type", Visibility: openapi.RootVisibilityExtended},
	{Name: "endpoint", Visibility: openapi.RootVisibilityExtended},
	{Name: "external-account-type", Visibility: openapi.RootVisibilityExtended},
	{Name: "auto-plugin-install", Visibility: openapi.RootVisibilityExtended},
	{Name: "auto-plugin-install-enable-pre", Visibility: openapi.RootVisibilityExtended},
	{Name: "bearer-token", Visibility: openapi.RootVisibilityExtended},
	{Name: "bearer-token-header-key", Visibility: openapi.RootVisibilityExtended},
	{Name: "secure", Visibility: openapi.RootVisibilityExtended},
	{Name: "force", Visibility: openapi.RootVisibilityExtended},
	{Name: "header", Visibility: openapi.RootVisibilityExtended},
	{Name: "body", Visibility: openapi.RootVisibilityExtended},
	{Name: "pager", Visibility: openapi.RootVisibilityExtended},
	{Name: "waiter", Visibility: openapi.RootVisibilityExtended},
	{Name: "dryrun", Visibility: openapi.RootVisibilityExtended},
	{Name: "estimate-cost", Visibility: openapi.RootVisibilityExtended},
	{Name: "estimate-cost-context", Visibility: openapi.RootVisibilityExtended},
	{Name: "quiet", Visibility: openapi.RootVisibilityExtended},
	{Name: "log-level", Visibility: openapi.RootVisibilityExtended},
	{Name: "method", Visibility: openapi.RootVisibilityExtended},
	{Name: "cli-section", Visibility: openapi.RootVisibilityExtended},
}

func newRootHelpInput(root *cli.Command, catalog *canonicalmeta.ProductsIndex) (openapi.RootHelpInput, error) {
	return openapi.BuildRootHelpInput(root, catalog, rootCommandHelpSpecs, rootFlagHelpSpecs)
}

// Resolve AI mode without loading the broken credential configuration or
// executing a command. Respect flag values and the positional terminator.
func startupAIMode(args []string) bool {
	flags := cli.NewFlagSet()
	config.AddFlags(flags)
	openapi.AddFlags(flags)
	forceOn, forceOff := false, false
	configDir := config.GetConfigPath()
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			break
		}
		name, value, inline := strings.Cut(arg, "=")
		if !strings.HasPrefix(name, "-") {
			continue
		}
		flag := flags.Get(strings.TrimPrefix(name, "--"))
		if flag == nil && len(name) == 2 && name[0] == '-' {
			flag = flags.GetByShorthand(rune(name[1]))
		}
		if flag == nil {
			continue
		}
		if flag.AssignedMode != cli.AssignedNone && !inline && i+1 < len(args) {
			i++
			value = args[i]
		}
		switch flag.Name {
		case "cli-ai-mode":
			forceOn = true
		case "no-cli-ai-mode":
			forceOff = true
		case "config-path":
			if value != "" {
				configDir = filepath.Dir(value)
			}
		}
	}
	cfg, _ := aimode.Load(configDir)
	if forceOff {
		return false
	}
	if forceOn {
		return true
	}
	if enabled, ok := aimode.EnvironmentOverride(); ok {
		return enabled
	}
	if util.DetectAgentName() != "" && aimode.AgentAIModeIntegrationEnabled() {
		return true
	}
	return aimode.EnabledForCommand(cfg, false, false)
}
