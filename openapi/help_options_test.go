package openapi

import (
	"io"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testHelpOptionsContext() *cli.Context {
	ctx := cli.NewCommandContext(io.Discard, io.Discard)
	AddFlags(ctx.Flags())
	return ctx
}

func assignHelpFlag(t *testing.T, flag *cli.Flag, value string) {
	t.Helper()
	require.NotNil(t, flag)
	flag.SetAssigned(true)
	flag.SetValue(value)
}

func TestParseHelpOptionsDefaultsToRequest(t *testing.T) {
	opts, err := parseHelpOptions(testHelpOptionsContext(), []string{"ecs", "DescribeInstances"})
	require.NoError(t, err)
	assert.Equal(t, helpSectionRequest, opts.Section)
	assert.False(t, opts.SectionExplicit)
	assert.Empty(t, opts.Search)
	assert.False(t, opts.All)
	assert.Equal(t, cli.HelpOutputText, opts.Output)
}

func TestParseHelpOptionsAcceptsResponseSearch(t *testing.T) {
	ctx := testHelpOptionsContext()
	assignHelpFlag(t, CliHelpSectionFlag(ctx.Flags()), " Response ")
	assignHelpFlag(t, CliHelpSearchFlag(ctx.Flags()), " instance-id ")

	opts, err := parseHelpOptions(ctx, []string{"ecs", "DescribeInstances"})
	require.NoError(t, err)
	assert.Equal(t, helpSectionResponse, opts.Section)
	assert.True(t, opts.SectionExplicit)
	assert.Equal(t, "instance-id", opts.Search)
}

func TestParseHelpOptionsValidatesScopeAndValues(t *testing.T) {
	tests := []struct {
		name   string
		target []string
		setup  func(*cli.FlagSet)
		want   string
	}{
		{
			name:   "invalid section",
			target: []string{"ecs", "DescribeInstances"},
			setup: func(fs *cli.FlagSet) {
				assignHelpFlag(t, CliHelpSectionFlag(fs), "headers")
			},
			want: "--cli-section must be request or response, got \"headers\"",
		},
		{
			name:   "section needs api",
			target: []string{"ecs"},
			setup: func(fs *cli.FlagSet) {
				assignHelpFlag(t, CliHelpSectionFlag(fs), "request")
			},
			want: "--cli-section requires a product and an API",
		},
		{
			name:   "search needs keyword",
			target: []string{"ecs"},
			setup: func(fs *cli.FlagSet) {
				assignHelpFlag(t, CliHelpSearchFlag(fs), "  ")
			},
			want: "--help-search requires a non-empty query",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := testHelpOptionsContext()
			tt.setup(ctx.Flags())
			_, err := parseHelpOptions(ctx, tt.target)
			assert.EqualError(t, err, tt.want)
		})
	}
}

func TestParseHelpOptionsAcceptsAllForAPI(t *testing.T) {
	ctx := testHelpOptionsContext()
	assignHelpFlag(t, CliHelpAllFlag(ctx.Flags()), "")

	opts, err := parseHelpOptions(ctx, []string{"ecs", "DescribeInstances"})
	require.NoError(t, err)
	assert.True(t, opts.All)
}

func TestParseHelpOptionsRejectsSearchWithAll(t *testing.T) {
	ctx := testHelpOptionsContext()
	assignHelpFlag(t, CliHelpSearchFlag(ctx.Flags()), "instance")
	assignHelpFlag(t, CliHelpAllFlag(ctx.Flags()), "")

	_, err := parseHelpOptions(ctx, []string{"ecs"})
	assert.EqualError(t, err, "--help-search conflicts with --help-all")
}

func TestParseHelpOptionsValidatesCLIOutput(t *testing.T) {
	ctx := testHelpOptionsContext()
	assignHelpFlag(t, CliOutputFlag(ctx.Flags()), "json")

	opts, err := parseHelpOptions(ctx, []string{"ecs"})
	require.NoError(t, err)
	assert.Equal(t, cli.HelpOutputJSON, opts.Output)
	assert.False(t, canonicalHelpOptionAssigned(ctx.Flags()), "--cli-output alone must not enter Help")

	assignHelpFlag(t, CliOutputFlag(ctx.Flags()), "yaml")
	_, err = parseHelpOptions(ctx, []string{"ecs"})
	assert.EqualError(t, err, "--cli-output only supports json, got \"yaml\"")
}

func TestCanonicalHelpOptionAssigned(t *testing.T) {
	ctx := testHelpOptionsContext()
	assert.False(t, canonicalHelpOptionAssigned(ctx.Flags()))
	assignHelpFlag(t, CliHelpSearchFlag(ctx.Flags()), "instance")
	assert.True(t, canonicalHelpOptionAssigned(ctx.Flags()))
}

func TestCommandoRejectsCanonicalHelpOptionsOutsideHelp(t *testing.T) {
	c, stdout, stderr := newTestCommando()
	ctx := cli.NewCommandContext(stdout, stderr)
	AddFlags(ctx.Flags())
	assignHelpFlag(t, CliHelpSectionFlag(ctx.Flags()), "response")

	err := c.main(ctx, []string{"ecs", "DescribeInstances"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "can only be used with `aliyun help ...` or --help")
}
