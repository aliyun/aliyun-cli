package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/bundledmeta"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	sysmock "github.com/aliyun/aliyun-cli/v3/sysconfig/mock"
	"github.com/stretchr/testify/require"
)

func TestMainEntryPointAndMetadataGenerationMode(t *testing.T) {
	clearAgentDetectionEnv(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOMEDRIVE", "")
	t.Setenv("HOMEPATH", "")
	t.Setenv(sysmock.EnvMockEnabled, "false")
	fixture, err := filepath.Abs("../canonicalmeta/testdata")
	require.NoError(t, err)
	oldMetadata, oldArgs, oldLanguage := bundledmeta.Metadatas, os.Args, i18n.GetLanguage()
	t.Cleanup(func() { bundledmeta.Metadatas, os.Args = oldMetadata, oldArgs; i18n.SetLanguage(oldLanguage) })
	bundledmeta.Metadatas = os.DirFS(fixture)
	t.Chdir(t.TempDir())
	var stdout, stderr bytes.Buffer
	resetMainHooks(t, &stdout, &stderr, nil)
	t.Setenv("GENERATE_METADATA", "YES")
	Main(nil)
	require.FileExists(t, filepath.Join("cli-metadata", "commands.json"))
	require.FileExists(t, filepath.Join("cli-metadata", "metadatas", "demo", "CreateReport.json"))
	t.Setenv("GENERATE_METADATA", "")
	os.Args = []string{"aliyun", "version"}
	stdout.Reset()
	main()
	require.Contains(t, stdout.String(), cli.GetVersion())
	require.Empty(t, stderr.String())
	require.True(t, startupAIMode([]string{"-p", "selected", "--cli-ai-mode"}))
	require.NotPanics(t, func() { prepareUtilityCommand(nil, true) })
}
