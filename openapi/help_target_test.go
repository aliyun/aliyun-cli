package openapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildHelpCommandUsesOneStyleAwareGrammar(t *testing.T) {
	tests := []struct {
		name   string
		target HelpTarget
		want   string
	}{
		{
			name:   "root default",
			target: HelpTarget{Level: HelpLevelRoot},
			want:   "aliyun --help",
		},
		{
			name: "root all json",
			target: HelpTarget{
				Level:     HelpLevelRoot,
				Operation: HelpOperationAll,
				Output:    HelpOutputJSON,
			},
			want: "aliyun --help-all --cli-output json",
		},
		{
			name: "camel product search preserves legacy version flag",
			target: HelpTarget{
				Level:        HelpLevelProduct,
				Product:      "ecs",
				CommandStyle: CommandStyleCamel,
				Version:      "2014-05-26",
				Operation:    HelpOperationSearch,
				SearchQuery:  "instance id",
			},
			want: "aliyun ecs --version 2014-05-26 --help-search 'instance id'",
		},
		{
			name: "kebab action uses api version flag",
			target: HelpTarget{
				Level:        HelpLevelAction,
				Product:      "ecs",
				Action:       "describe-instances",
				CommandStyle: CommandStyleKebab,
				Version:      "2014-05-26",
			},
			want: "aliyun ecs describe-instances --api-version 2014-05-26 --help",
		},
		{
			name: "explicit version spelling wins",
			target: HelpTarget{
				Level:        HelpLevelAction,
				Product:      "ecs",
				Action:       "DescribeInstances",
				CommandStyle: CommandStyleCamel,
				VersionFlag:  VersionFlagAPI,
				Version:      "2014-05-26",
			},
			want: "aliyun ecs DescribeInstances --api-version 2014-05-26 --help",
		},
		{
			name: "parameter help is suffix form",
			target: HelpTarget{
				Level:        HelpLevelParameter,
				Product:      "ecs",
				Action:       "DescribeInstances",
				Parameter:    "--InstanceIds",
				CommandStyle: CommandStyleCamel,
				Version:      "2014-05-26",
			},
			want: "aliyun ecs DescribeInstances --version 2014-05-26 --InstanceIds --help",
		},
		{
			name: "response section is prefix form",
			target: HelpTarget{
				Level:           HelpLevelAction,
				Product:         "ecs",
				Action:          "DescribeInstances",
				CommandStyle:    CommandStyleCamel,
				Version:         "2014-05-26",
				Section:         HelpSectionResponse,
				SectionExplicit: true,
				Output:          HelpOutputJSON,
			},
			want: "aliyun help ecs DescribeInstances --version 2014-05-26 --cli-section response --cli-output json",
		},
		{
			name: "request section search is prefix form",
			target: HelpTarget{
				Level:           HelpLevelAction,
				Product:         "ecs",
				Action:          "describe-instances",
				CommandStyle:    CommandStyleKebab,
				Section:         HelpSectionRequest,
				SectionExplicit: true,
				Operation:       HelpOperationSearch,
				SearchQuery:     "instance",
			},
			want: "aliyun help ecs describe-instances --cli-section request --help-search instance",
		},
		{
			name: "utility help uses canonical subtree",
			target: HelpTarget{
				Level:   HelpLevelUtility,
				Utility: "mcp-proxy",
			},
			want: "aliyun utils mcp-proxy --help",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildHelpCommand(tt.target)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildHelpCommandRejectsInvalidTargets(t *testing.T) {
	tests := []struct {
		name   string
		target HelpTarget
	}{
		{
			name:   "search requires query",
			target: HelpTarget{Level: HelpLevelProduct, Product: "ecs", Operation: HelpOperationSearch},
		},
		{
			name:   "action requires action",
			target: HelpTarget{Level: HelpLevelAction, Product: "ecs", CommandStyle: CommandStyleCamel},
		},
		{
			name: "section does not support all",
			target: HelpTarget{
				Level: HelpLevelAction, Product: "ecs", Action: "DescribeInstances",
				CommandStyle: CommandStyleCamel, Section: HelpSectionRequest,
				SectionExplicit: true, Operation: HelpOperationAll,
			},
		},
		{
			name:   "unknown output",
			target: HelpTarget{Level: HelpLevelRoot, Output: HelpOutput("yaml")},
		},
		{
			name: "invalid version flag",
			target: HelpTarget{
				Level: HelpLevelProduct, Product: "ecs", CommandStyle: CommandStyleCamel,
				Version: "2014-05-26", VersionFlag: APIVersionFlag("revision"),
			},
		},
		{
			name: "invalid command style",
			target: HelpTarget{
				Level: HelpLevelProduct, Product: "ecs", CommandStyle: CommandStyle("snake"),
			},
		},
		{
			name: "response section cannot be silently ignored",
			target: HelpTarget{
				Level: HelpLevelAction, Product: "ecs", Action: "DescribeInstances",
				CommandStyle: CommandStyleCamel, Section: HelpSectionResponse,
			},
		},
		{
			name: "assigned parameter is not parameter Help",
			target: HelpTarget{
				Level: HelpLevelParameter, Product: "ecs", Action: "DescribeInstances",
				CommandStyle: CommandStyleCamel, Parameter: "InstanceId=i-123",
			},
		},
		{
			name: "command token cannot inject shell syntax",
			target: HelpTarget{
				Level: HelpLevelProduct, Product: "ecs;exit", CommandStyle: CommandStyleCamel,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := BuildHelpCommand(tt.target)
			require.Error(t, err)
		})
	}
}

func TestBuildHelpCommandAppendsAllToSearch(t *testing.T) {
	target := HelpTarget{
		Level:       HelpLevelProduct,
		Product:     "vpc",
		Operation:   HelpOperationSearch,
		SearchQuery: "instance",
		SearchAll:   true,
	}
	command, err := BuildHelpCommand(target)
	require.NoError(t, err)
	assert.Equal(t, "aliyun vpc --help-search instance --help-all", command)
}
