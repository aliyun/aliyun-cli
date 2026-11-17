package cli

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/stretchr/testify/require"
)

var defaultCompletionBinaryHook = hookGetBinaryPath

func TestRepeatableFlagFieldsAndInvalidDefinitions(t *testing.T) {
	flag := &Flag{Name: "mapping", AssignedMode: AssignedRepeatable, Fields: []Field{{Key: "name"}, {Key: ""}}}
	require.NoError(t, flag.assign("name=first"))
	require.NoError(t, flag.assign("=plain"))
	value, ok := flag.GetFieldValue("name")
	require.True(t, ok)
	require.Equal(t, "first", value)
	value, ok = flag.GetFieldValue("")
	require.True(t, ok)
	require.Equal(t, "plain", value)
	require.PanicsWithError(t, "unexpected Flag.AssignedMode 123", func() { (&Flag{AssignedMode: 123}).needValue() })
	require.PanicsWithError(t, "flag invalid with fields must use AssignedRepeatable", func() {
		(&Flag{Name: "invalid", AssignedMode: AssignedOnce, Fields: []Field{{Key: "name"}}}).checkValid()
	})
	var optional *Flag
	require.False(t, optional.IsAssigned())
	value, ok = optional.GetValue()
	require.False(t, ok)
	require.Empty(t, value)
	require.Equal(t, 10, optional.GetIntegerOrDefault(10))
	var absent *FlagSet
	require.Nil(t, absent.Get("missing"))
}

func TestCommandParsingAndValidationErrors(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"--unknown"}, "invalid flag"},
		{[]string{"argument", "--value", "--required", "set"}, "must be assigned with value"},
		{[]string{"argument"}, "required"},
	} {
		ctx := NewCommandContext(io.Discard, io.Discard)
		command := &Command{Name: "sample", Run: func(*Context, []string) error { t.Fatal("invalid input reached Run"); return nil }}
		command.Flags().Add(&Flag{Name: "value", AssignedMode: AssignedOnce})
		command.Flags().Add(&Flag{Name: "required", Required: true, AssignedMode: AssignedOnce})
		ctx.EnterCommand(command)
		require.ErrorContains(t, command.executeInner(ctx, tc.args), tc.want)
	}
	for _, args := range [][]string{{"--help"}, {"argument", "help"}} {
		ctx := NewCommandContext(io.Discard, io.Discard)
		command := &Command{Name: "sample", Short: i18n.T("sample command", "示例命令"), Run: func(*Context, []string) error { t.Fatal("help reached Run"); return nil }}
		ctx.EnterCommand(command)
		require.NoError(t, command.executeInner(ctx, args))
		require.True(t, ctx.help)
	}
	ctx := NewCommandContext(io.Discard, io.Discard)
	ctx.EnterCommand(&Command{Name: "unknown", EnableUnknownFlag: true})
	ctx.hasPluginSubCmdSet = true
	ctx.hasPluginSubCmd = false
	_, err := ctx.detectFlag("custom")
	require.NoError(t, err)
	_, err = ctx.detectFlag("custom")
	require.Error(t, err)
	_, err = ctx.detectFlagByShorthand('x')
	require.NoError(t, err)
	_, err = ctx.detectFlagByShorthand('x')
	require.Error(t, err)
}

func TestCompletionCommandRoundTripAndInstallFailures(t *testing.T) {
	home := t.TempDir()
	t.Setenv("MOCK_USER_HOME_DIR", home)
	rc := filepath.Join(home, ".bashrc")
	zrc := filepath.Join(home, ".zshrc")
	require.NoError(t, os.WriteFile(rc, []byte("existing bash\n"), 0600))
	require.NoError(t, os.WriteFile(zrc, []byte("autoload -U +X bashcompinit && bashcompinit -i\n"), 0600))
	oldHook, oldAssigned := hookGetBinaryPath, uninstallFlag.IsAssigned()
	t.Cleanup(func() { hookGetBinaryPath = oldHook; uninstallFlag.SetAssigned(oldAssigned) })
	hookGetBinaryPath = defaultCompletionBinaryHook
	var out bytes.Buffer
	ctx := NewCommandContext(io.Discard, &out)
	uninstallFlag.SetAssigned(false)
	require.NoError(t, NewAutoCompleteCommand().Run(ctx, nil))
	bin, err := getBinaryPath()
	require.NoError(t, err)
	require.True(t, lineInFile(rc, (bashInstaller{rc}).cmd("aliyun", bin)))
	require.True(t, lineInFile(zrc, (zshInstaller{zrc}).cmd("aliyun", bin)))
	uninstallFlag.SetAssigned(true)
	require.NoError(t, NewAutoCompleteCommand().Run(ctx, nil))
	require.False(t, lineInFile(rc, (bashInstaller{rc}).cmd("aliyun", bin)))
	require.False(t, lineInFile(zrc, (zshInstaller{zrc}).cmd("aliyun", bin)))
	uninstallCompletion(ctx, "aliyun")
	require.Contains(t, out.String(), "uninstall")
	require.NoError(t, os.Remove(rc))
	require.NoError(t, os.Mkdir(rc, 0700))
	out.Reset()
	installCompletion(ctx, "aliyun")
	require.Contains(t, out.String(), "install completion failed")
}

func TestCompletionTemporaryFileFailuresPreserveOriginal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rc")
	require.NoError(t, os.WriteFile(path, []byte("keep\nremove\n"), 0600))
	missing := filepath.Join(t.TempDir(), "missing")
	for _, name := range []string{"TMPDIR", "TMP", "TEMP"} {
		t.Setenv(name, missing)
	}
	_, err := removeContentToTempFile(path, "remove")
	require.Error(t, err)
	require.Error(t, removeFromFile(path, "remove"))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "keep\nremove\n", string(data))
}

func TestErrorEnvelopeNormalizesDuplicateAndEmptySuggestions(t *testing.T) {
	cause := errors.New("original failure")
	require.Equal(t, cause.Error(), (&AgentError{cause: cause}).Error())
	require.Empty(t, (&AgentError{}).Error())
	require.False(t, IsAIRecoveryEligible(nil))
	envelope := AgentErrorEnvelope{Message: "invalid input", DidYouMean: []string{" region ", "region", " "}, Recovery: AgentErrorRecovery{Action: "inspect_help", Hint: "inspect parameters"}}
	err := NewAgentError(envelope, cause)
	require.NotNil(t, err)
	require.Equal(t, []string{"region"}, err.Envelope().DidYouMean)
	envelope.DidYouMean = []string{" ", "\t"}
	err = NewAgentError(envelope, cause)
	require.Nil(t, err.Envelope().DidYouMean)
	require.Equal(t, "none", (&Parser{}).flagOrigin("plain", nil))
	parser, flags := newTestParser("--test", "value", "--prev")
	_, parseErr := parser.ReadAll()
	require.NoError(t, parseErr)
	require.True(t, flags.Get("prev").IsAssigned())
	parser.currentFlag = &Flag{AssignedMode: AssignedNone}
	parser.currentFlagOrigin = "long_flag_candidate"
	require.False(t, parser.canConsumeDashLeadingValue("-1"))
	parser.currentFlag.dynamicUnknown = true
	require.False(t, parser.canConsumeDashLeadingValue("-1"))
}

func TestCompletionReadErrorsDoNotChangeInput(t *testing.T) {
	temporary := t.TempDir()
	for _, name := range []string{"TMPDIR", "TMP", "TEMP"} {
		t.Setenv(name, temporary)
	}
	source := t.TempDir()
	require.False(t, lineInFile(filepath.Join(source, "missing"), "line"))
	_, err := removeContentToTempFile(source, "line")
	require.Error(t, err)
	require.Error(t, copyFile(source, filepath.Join(temporary, "copy")))
	require.DirExists(t, source)
}

func TestSuggestionsDiscardWeakerMatchesAndIgnoreEmptyNames(t *testing.T) {
	suggester := NewSuggester("cat", 2)
	suggester.Apply("bat")
	suggester.Apply("cat")
	require.Equal(t, []string{"cat"}, suggester.GetResults())
	names, total := PrefixSuggestions("get", []string{"", " ", "GetInstance"}, 5)
	require.Equal(t, []string{"GetInstance"}, names)
	require.Equal(t, 1, total)
	options := DefaultOptions
	options.SubCost = 1
	require.Equal(t, EditScript{Sub}, EditScriptForStrings([]rune("a"), []rune("b"), options))
	flags := NewFlagSet()
	flag := &Flag{Name: "shared", Persistent: true}
	flags.Add(flag)
	merged := flags.mergeWith(flags, func(*Flag) bool { return true })
	require.Equal(t, []*Flag{flag}, merged.Flags())
	var out bytes.Buffer
	ctx := NewCommandContext(&out, io.Discard)
	command := &Command{Name: "root"}
	command.AddSubCommand(&Command{Name: "hidden", Hidden: true, Short: i18n.T("hidden", "")})
	command.AddSubCommand(&Command{Name: "visible", Short: i18n.T("visible", "")})
	command.PrintSubCommands(ctx)
	require.Contains(t, out.String(), "visible")
	require.NotContains(t, out.String(), "hidden")
}
