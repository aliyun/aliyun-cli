package cli

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseHelpOptionsRecognizesFinalSurface(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want HelpOptions
	}{
		{
			name: "ordinary arguments do not request help",
			args: []string{"ecs", "DescribeInstances", "--cli-output", "json"},
			want: HelpOptions{
				Operation: HelpOperationDefault,
				Output:    HelpOutputJSON,
				Section:   HelpSectionRequest,
			},
		},
		{
			name: "default help",
			args: []string{"ecs", "--help"},
			want: HelpOptions{
				Requested: true,
				Operation: HelpOperationDefault,
				Output:    HelpOutputText,
				Section:   HelpSectionRequest,
			},
		},
		{
			name: "compatible help command",
			args: []string{"help", "ecs"},
			want: HelpOptions{
				Requested: true,
				Operation: HelpOperationDefault,
				Output:    HelpOutputText,
				Section:   HelpSectionRequest,
			},
		},
		{
			name: "all and json",
			args: []string{"--help-all", "--cli-output=json"},
			want: HelpOptions{
				Requested: true,
				Operation: HelpOperationAll,
				Output:    HelpOutputJSON,
				Section:   HelpSectionRequest,
			},
		},
		{
			name: "search and response section",
			args: []string{"help", "ecs", "DescribeInstances", "--cli-section", " Response ", "--help-search= instance id "},
			want: HelpOptions{
				Requested:       true,
				Operation:       HelpOperationSearch,
				SearchQuery:     "instance id",
				SearchAll:       true,
				Output:          HelpOutputText,
				Section:         HelpSectionResponse,
				SectionExplicit: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseHelpOptions(tt.args)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseHelpOptionsComposesSearchWithAll(t *testing.T) {
	orders := [][]string{
		{"--help-search", "instance", "--help-all"},
		{"--help-all", "--help-search=instance"},
	}
	for _, args := range orders {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			opts, err := ParseHelpOptions(args)
			require.NoError(t, err)
			assert.Equal(t, HelpOperationSearch, opts.Operation)
			assert.Equal(t, "instance", opts.SearchQuery)
			assert.True(t, opts.SearchAll)
			assert.True(t, opts.Requested)
		})
	}
}

func TestParseHelpOptionsSearchIsEquivalentToSearchAll(t *testing.T) {
	search, err := ParseHelpOptions([]string{"--help-search", "instance"})
	require.NoError(t, err)
	searchAll, err := ParseHelpOptions([]string{"--help-search", "instance", "--help-all"})
	require.NoError(t, err)
	assert.Equal(t, searchAll, search)
}

func TestParseHelpOptionsRejectsInvalidCombinations(t *testing.T) {
	tests := []struct {
		name string
		args []string
		code HelpOptionErrorCode
	}{
		{name: "duplicate default", args: []string{"--help", "-h"}, code: HelpOptionDuplicate},
		{name: "duplicate all", args: []string{"--help-all", "--help-all"}, code: HelpOptionDuplicate},
		{name: "duplicate all after search", args: []string{"--help-search", "instance", "--help-all", "--help-all"}, code: HelpOptionDuplicate},
		{name: "conflicting operations", args: []string{"--help", "--help-search", "instance"}, code: HelpOptionConflict},
		{name: "empty search separated", args: []string{"--help-search", "  "}, code: HelpOptionEmptySearch},
		{name: "empty search equals", args: []string{"--help-search="}, code: HelpOptionEmptySearch},
		{name: "search cannot consume another option", args: []string{"--help-search", "--cli-output", "json"}, code: HelpOptionEmptySearch},
		{name: "invalid output", args: []string{"--help", "--cli-output", "yaml"}, code: HelpOptionInvalidOutput},
		{name: "duplicate output", args: []string{"--cli-output", "json", "--cli-output=json"}, code: HelpOptionDuplicate},
		{name: "invalid section", args: []string{"help", "ecs", "DescribeInstances", "--cli-section", "headers"}, code: HelpOptionInvalidSection},
		{name: "duplicate section", args: []string{"--cli-section=request", "--cli-section", "response"}, code: HelpOptionDuplicate},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseHelpOptions(tt.args)
			require.Error(t, err)
			var optionErr *HelpOptionError
			require.True(t, errors.As(err, &optionErr), "error = %T %v", err, err)
			assert.Equal(t, tt.code, optionErr.Code)
		})
	}
}
