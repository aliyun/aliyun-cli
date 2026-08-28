package main

import (
	"strings"

	"github.com/aliyun/aliyun-cli/v3/cli"
	go_migrate "github.com/aliyun/aliyun-cli/v3/go-migrate"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/aliyun/aliyun-cli/v3/mcpproxy"
	"github.com/aliyun/aliyun-cli/v3/openapi"
)

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
