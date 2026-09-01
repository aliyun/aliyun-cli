package cli

import (
	"bytes"
	"errors"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandSubCommandNamesAndMetadata(t *testing.T) {
	var nilCommand *Command
	assert.Nil(t, nilCommand.SubCommandNames())
	root := &Command{
		Name: "aliyun", Short: i18n.T("root short", "根命令"), Long: i18n.T("root long", "根说明"),
		Usage: "aliyun <command>", Sample: "aliyun configure", Hidden: false,
	}
	assert.Nil(t, root.SubCommandNames())
	child := &Command{Name: "demo", Short: i18n.T("demo short", "演示")}
	root.AddSubCommand(child)
	root.subCommands = append(root.subCommands, nil)
	assert.Equal(t, []string{"demo"}, root.SubCommandNames())

	flag := &Flag{
		Name: "profile", Shorthand: 'p', Short: i18n.T("profile short", "配置短说明"), Long: i18n.T("profile long", "配置长说明"),
		DefaultValue: "default", Required: true, Aliases: []string{"account"}, AssignedMode: AssignedOnce,
		Persistent: true, Hidden: true, Category: "global",
	}
	root.Flags().Add(flag)
	// Metadata recursion expects actual child commands; keep the nil entry only
	// for the SubCommandNames nil-filter contract above.
	root.subCommands = root.subCommands[:1]
	metadata := map[string]*Metadata{}
	root.GetMetadata(metadata)
	require.Contains(t, metadata, "aliyun")
	require.Contains(t, metadata, "aliyun demo")
	rootMetadata := metadata["aliyun"]
	assert.Equal(t, "root long", rootMetadata.Long["en"])
	require.Contains(t, rootMetadata.Flags, "profile")
	profile := rootMetadata.Flags["profile"]
	assert.Equal(t, 'p', profile.Shorthand)
	assert.Equal(t, "default", profile.DefaultValue)
	assert.True(t, profile.Required)
	assert.True(t, profile.Persistent)
	assert.True(t, profile.Hidden)
	assert.Equal(t, "global", profile.Category)
}

func TestContextRuntimeSettingsAccessors(t *testing.T) {
	ctx := NewCommandContext(&bytes.Buffer{}, &bytes.Buffer{})
	assert.False(t, ctx.Insecure())
	ctx.SetInsecure(true)
	assert.True(t, ctx.Insecure())
	envs := map[string]string{"LANG": "en"}
	ctx.SetRuntimeEnvs(envs)
	assert.Equal(t, envs, ctx.GetRuntimeEnvs())
}

func TestErrorAgentContractsAndDefaultWriters(t *testing.T) {
	invalidCommand := &InvalidCommandError{Name: "missing"}
	invalidCommand.AIRecoveryEligible()
	assert.Nil(t, invalidCommand.GetSuggestions())
	assert.Equal(t, "Use `--help` for more information.", invalidCommand.GetTip("en"))

	invalidFlag := &InvalidFlagError{Flag: "--missing"}
	invalidFlag.AIRecoveryEligible()
	assert.Equal(t, "invalid flag --missing", invalidFlag.AgentMessage())
	assert.Equal(t, "aliyun help", invalidFlag.AgentHelpCommand())
	assert.Nil(t, invalidFlag.AgentSuggestions())

	option := &HelpOptionError{Code: HelpOptionErrorCode("future")}
	assert.Equal(t, "invalid Help options", option.Error())
	assert.Nil(t, option.GetSuggestions())
	option.AIRecoveryEligible()

	assert.NotNil(t, DefaultStdoutWriter())
	assert.NotNil(t, DefaultStderrWriter())
	assert.False(t, IsAIRecoveryEligible(errors.New("plain")))
}
