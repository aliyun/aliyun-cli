package openapi

import (
	"io"
	"testing"
	"testing/fstest"

	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/stretchr/testify/require"
)

func TestRecoverySearchDeclinesUnavailableTargets(t *testing.T) {
	for _, request := range []RecoverySearchRequest{
		{},
		{Product: "absent", Keyword: "id"},
		{Product: "demo", Version: "missing", Keyword: "id"},
		{Product: "demo", API: "Absent", Keyword: "id"},
		{Product: "demo", API: "Absent", Section: "response", Keyword: "id"},
	} {
		c, ctx, _, _ := newCanonicalHelpTestContext(t)
		require.False(t, c.validateRecoverySearch(ctx, request))
	}
	c, _, _, _ := newCanonicalHelpTestContext(t)
	require.False(t, c.validateRecoverySearch(nil, RecoverySearchRequest{Keyword: "id"}))
	require.False(t, c.validateRecoverySearch(cli.NewCommandContext(io.Discard, io.Discard), RecoverySearchRequest{Keyword: "id"}))
	c.library.helpRepo = nil
	require.False(t, c.validateRecoverySearch(nil, RecoverySearchRequest{Keyword: "id"}))
	require.False(t, (*Commando)(nil).validateRecoverySearch(nil, RecoverySearchRequest{Keyword: "id"}))
}

func TestLegacyHelpReportsMetadataAndOutputFailures(t *testing.T) {
	for _, target := range []string{"root", "product", "response", "unknown product", "unknown API", "missing response repository", "malformed catalog", "invalid help options"} {
		t.Run(target, func(t *testing.T) {
			c, original, _, _ := newCanonicalHelpTestContext(t)
			ctx := cli.NewCommandContext(&endpointFailWriter{}, io.Discard)
			ctx.EnterCommand(original.Command())
			CliHelpAllFlag(ctx.Flags()).SetAssigned(true)
			args := []string(nil)
			want := io.ErrClosedPipe.Error()
			switch target {
			case "product":
				args = []string{"demo"}
			case "response", "missing response repository":
				args = []string{"demo", "CreateReport"}
				CliHelpSectionFlag(ctx.Flags()).SetValue("response")
				CliHelpSectionFlag(ctx.Flags()).SetAssigned(true)
				if target == "missing response repository" {
					c.library.helpRepo = nil
					want = "repository is unavailable"
				}
			case "unknown product":
				args, want = []string{"absent"}, "absent"
			case "unknown API":
				args, want = []string{"demo", "Absent"}, "Absent"
			case "malformed catalog":
				c.library.helpRepo = canonicalmeta.NewRepository(fstest.MapFS{"metadatas/products.json": {Data: []byte("{")}})
				want = "parse"
			case "invalid help options":
				CliHelpSectionFlag(ctx.Flags()).SetAssigned(true)
				CliHelpSectionFlag(ctx.Flags()).SetValue("invalid")
				want = "invalid"
			}
			require.ErrorContains(t, c.legacyHelp(ctx, args), want)
		})
	}
}

func TestHostHelpRefusesMissingRepositoryAndInvalidParameter(t *testing.T) {
	c, ctx, _, _ := newCanonicalHelpTestContext(t)
	target := HelpTarget{Level: HelpLevelParameter, Product: "demo", Action: "CreateReport", Parameter: "missing", CommandStyle: CommandStyleCamel, Operation: HelpOperationDefault, Output: HelpOutputText, Provider: HelpProviderHost}
	require.ErrorContains(t, c.renderHostHelpTarget(ctx, target, false), "missing")
	target.Output = HelpOutputJSON
	require.Error(t, c.renderHostHelpTarget(ctx, target, false))
	c.library.helpRepo = nil
	require.ErrorContains(t, c.renderHostHelpTarget(ctx, target, false), "metadata repository is unavailable")
}
