package mock

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMockStorageRejectsUnreadableAndInvalidRecords(t *testing.T) {
	dir := t.TempDir()
	_, err := Load(dir)
	require.Error(t, err)
	require.Empty(t, LoadLenient(dir))
	path := filepath.Join(t.TempDir(), "records.json")
	require.NoError(t, os.WriteFile(path, []byte("null"), 0600))
	require.Empty(t, LoadLenient(path))
	require.NoError(t, os.WriteFile(path, []byte(`[{"name":"invalid","cmd":"ecs List","exit_code":0,"stdout":"","stderr":"","times":-1}]`), 0600))
	require.Empty(t, LoadLenient(path))
	require.Error(t, Append(path, []Record{{Name: "new", Cmd: "ecs List"}}))
	_, err = DecodeInput([]byte(`[{"name":"invalid","cmd":"ecs List","exit_code":0,"stdout":"","stderr":"","times":-1}]`))
	require.Error(t, err)
	require.NoError(t, Save(path, nil))
	loaded, err := Load(path)
	require.NoError(t, err)
	require.Empty(t, loaded)
	require.Error(t, Save(dir, nil))
	require.NoError(t, os.WriteFile(path, []byte("file"), 0600))
	require.Error(t, Save(filepath.Join(path, "child"), nil))
}

func TestCommandMatchingEmptyRulesAndIdentityBoundaries(t *testing.T) {
	require.True(t, MatchCommand("", nil))
	require.False(t, MatchCommand("", []string{"ecs"}))
	require.False(t, MatchCommand("ecs ListInstances", []string{"ecs"}))
	require.False(t, MatchCommand("ecs ListInstances --RegionId cn-hangzhou", []string{"ecs", "ListInstances", "extra", "--RegionId", "cn-hangzhou"}))
	require.False(t, MatchCommand("ecs ListInstances --RegionId cn-hangzhou", []string{"ecs", "ListInstances", "--RegionId"}))
	require.Nil(t, StripLeadingGlobalFlags([]string{"--"}))
	groups := []parameterGroup{{tokens: []string{"--region", "cn-hangzhou"}}}
	require.True(t, matchParameterGroups(groups, []string{"--region", "cn-hangzhou"}, false))
	require.False(t, matchParameterGroups(groups, []string{"--region", "cn-hangzhou", "extra"}, false))
	require.True(t, matchParameterGroups(nil, nil, false))
}
