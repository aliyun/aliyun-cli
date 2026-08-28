package main

import (
	"bytes"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRootHelpInputDeclaresStableGroupsAndFlags(t *testing.T) {
	root := newRootCommand(config.NewProfile("default"), &bytes.Buffer{})
	input, err := newRootHelpInput(root, &canonicalmeta.ProductsIndex{Products: []canonicalmeta.ProductEntry{
		{Code: "Ecs", Name: map[string]string{"en": "Elastic Compute Service"}, Distribution: "openapi|go"},
	}})
	require.NoError(t, err)

	assert.Equal(t, []string{
		"configure", "plugin", "upgrade", "version", "auto-completion", "mock", "utils",
	}, rootHelpNamesByGroup(input.Commands, openapi.RootGroupCore))
	assert.Equal(t, []string{
		"utils list-supported-pricing-apis", "utils mcp-proxy", "utils go-migrate",
	}, rootHelpNamesByGroup(input.Commands, openapi.RootGroupUtils))
	assert.NotEmpty(t, rootHelpNamesByGroup(input.Commands, openapi.RootGroupExtension))
	assert.NotContains(t, rootHelpNamesByGroup(input.Commands, openapi.RootGroupExtension), "mcp-proxy")

	assert.Equal(t, []string{
		"profile", "region", "language", "version", "output", "cli-query", "cli-output",
		"dryrun", "yes", "cli-ai-mode", "help", "help-all", "help-search",
	}, rootHelpFlagNames(input.GlobalFlags, openapi.RootVisibilityDefault))
	assert.Equal(t, []string{"api-version"}, input.GlobalFlags[3].Aliases)
	assert.Equal(t, []string{"ecs"}, []string{input.Products[0].Code})
}

func TestNewRootHelpInputClassifiesEveryPublicRootFlag(t *testing.T) {
	root := newRootCommand(config.NewProfile("default"), &bytes.Buffer{})
	input, err := newRootHelpInput(root, &canonicalmeta.ProductsIndex{})
	require.NoError(t, err)

	public := []string{"help"}
	for _, flag := range root.Flags().Flags() {
		if !flag.Hidden {
			public = append(public, flag.Name)
		}
	}
	classified := make([]string, 0, len(input.GlobalFlags))
	for _, flag := range input.GlobalFlags {
		classified = append(classified, flag.Name)
	}
	assert.ElementsMatch(t, public, classified)
}

func rootHelpNamesByGroup(commands []openapi.RootCommandInput, group openapi.RootHelpGroup) []string {
	var names []string
	for _, command := range commands {
		if command.Group == group {
			names = append(names, command.Name)
		}
	}
	return names
}

func rootHelpFlagNames(flags []openapi.RootFlagInput, visibility openapi.RootVisibility) []string {
	var names []string
	for _, flag := range flags {
		if flag.Visibility == visibility {
			names = append(names, flag.Name)
		}
	}
	return names
}
