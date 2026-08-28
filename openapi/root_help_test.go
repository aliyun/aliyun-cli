package openapi

import (
	"testing"

	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildRootHelpInputUsesExplicitOfflineGroups(t *testing.T) {
	root := &cli.Command{Name: "aliyun", Short: i18n.T("Alibaba Cloud CLI", "阿里云 CLI")}
	configure := &cli.Command{Name: "configure", Short: i18n.T("Configure credentials", "配置凭证")}
	utils := &cli.Command{Name: "utils", Short: i18n.T("Local utilities", "本地工具")}
	mcpProxy := &cli.Command{Name: "mcp-proxy", Short: i18n.T("Start MCP proxy", "启动 MCP 代理")}
	utils.AddSubCommand(mcpProxy)
	root.AddSubCommand(configure)
	root.AddSubCommand(utils)
	root.AddSubCommand(&cli.Command{Name: "mcp-proxy", Hidden: true, Short: i18n.T("legacy alias", "旧别名")})
	root.Flags().Add(&cli.Flag{Name: "profile", Shorthand: 'p', Short: i18n.T("profile", "配置"), AssignedMode: cli.AssignedOnce})
	root.Flags().Add(&cli.Flag{Name: "internal", Hidden: true, Short: i18n.T("internal", "内部")})

	catalog := &canonicalmeta.ProductsIndex{Products: []canonicalmeta.ProductEntry{
		{Code: "VPC", Name: map[string]string{"en": "Virtual Private Cloud", "zh": "专有网络"}, Distribution: "openapi|go"},
		{Code: "ecs", Name: map[string]string{"en": "Elastic Compute Service", "zh": "云服务器"}, Distribution: "openapi"},
	}}

	input, err := BuildRootHelpInput(root, catalog, []RootCommandSpec{
		{Path: []string{"configure"}, Group: RootGroupCore},
		{Path: []string{"utils"}, Group: RootGroupCore},
		{Path: []string{"utils", "mcp-proxy"}, Group: RootGroupUtils, Aliases: []string{"mcp-proxy"}},
	}, []RootFlagSpec{
		{Name: "profile", Visibility: RootVisibilityDefault},
		{Name: "help", Visibility: RootVisibilityDefault},
	})
	require.NoError(t, err)

	assert.Equal(t, "aliyun", input.Name)
	assert.Equal(t, "Alibaba Cloud CLI", input.Description.EN)
	require.Len(t, input.Commands, 3)
	assert.Equal(t, RootCommandInput{
		Group:       RootGroupCore,
		Path:        []string{"configure"},
		Name:        "configure",
		Description: RootLocalizedText{EN: "Configure credentials", ZH: "配置凭证"},
	}, input.Commands[0])
	assert.Equal(t, []string{"mcp-proxy"}, input.Commands[2].Aliases)
	assert.Equal(t, RootGroupUtils, input.Commands[2].Group)
	assert.Equal(t, "utils mcp-proxy", input.Commands[2].Name)
	assert.NotContains(t, commandInputNames(input.Commands), "mcp-proxy", "hidden root alias must not be emitted as a root command")

	require.Len(t, input.GlobalFlags, 2)
	assert.Equal(t, "profile", input.GlobalFlags[0].Name)
	assert.Equal(t, 'p', input.GlobalFlags[0].Shorthand)
	assert.Equal(t, "help", input.GlobalFlags[1].Name)
	assert.Equal(t, 'h', input.GlobalFlags[1].Shorthand)
	assert.NotContains(t, rootFlagInputNames(input.GlobalFlags), "internal")

	require.Len(t, input.Products, 2)
	assert.Equal(t, []string{"ecs", "vpc"}, []string{input.Products[0].Code, input.Products[1].Code})
	assert.Equal(t, RootGroupProduct, input.Products[0].Group)
	assert.Equal(t, "openapi|go", input.Products[1].Distribution)
}

func TestBuildRootHelpInputRejectsImplicitCommandClassification(t *testing.T) {
	root := &cli.Command{Name: "aliyun", Short: i18n.T("CLI", "CLI")}
	root.AddSubCommand(&cli.Command{Name: "configure", Short: i18n.T("Configure", "配置")})
	root.AddSubCommand(&cli.Command{Name: "plugin", Short: i18n.T("Plugin", "插件")})

	_, err := BuildRootHelpInput(root, &canonicalmeta.ProductsIndex{}, []RootCommandSpec{
		{Path: []string{"configure"}, Group: RootGroupCore},
	}, nil)
	assert.EqualError(t, err, "visible root command \"plugin\" has no explicit Help group")
}

func TestBuildRootHelpInputRejectsImplicitFlagClassification(t *testing.T) {
	root := &cli.Command{Name: "aliyun", Short: i18n.T("CLI", "CLI")}
	root.Flags().Add(&cli.Flag{Name: "profile", Short: i18n.T("Profile", "配置")})
	root.Flags().Add(&cli.Flag{Name: "region", Short: i18n.T("Region", "地域")})

	_, err := BuildRootHelpInput(root, &canonicalmeta.ProductsIndex{}, nil, []RootFlagSpec{
		{Name: "profile", Visibility: RootVisibilityDefault},
		{Name: "help", Visibility: RootVisibilityDefault},
	})
	assert.EqualError(t, err, "visible root flag \"region\" has no explicit Help visibility")
}

func commandInputNames(commands []RootCommandInput) []string {
	names := make([]string, 0, len(commands))
	for _, command := range commands {
		names = append(names, command.Name)
	}
	return names
}

func rootFlagInputNames(flags []RootFlagInput) []string {
	names := make([]string, 0, len(flags))
	for _, flag := range flags {
		names = append(names, flag.Name)
	}
	return names
}
