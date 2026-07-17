// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"

	"github.com/stretchr/testify/assert"
)

func TestNewConfiguration(t *testing.T) {
	excf := &Configuration{
		CurrentProfile: DefaultConfigProfileName,
		Profiles: []Profile{
			NewProfile(DefaultConfigProfileName),
		},
	}
	cf := NewConfiguration()
	assert.Equal(t, excf, cf)
}

func TestCFNewProfile(t *testing.T) {
	cf := Configuration{
		CurrentProfile: "",
	}
	assert.Len(t, cf.Profiles, 0)
	p := cf.NewProfile("default")
	assert.Len(t, cf.Profiles, 1)
	exp := Profile{
		Name:         "default",
		Mode:         "",
		OutputFormat: "json",
		Language:     "en",
	}
	assert.Equal(t, exp, p)
}

func TestConfiguration(t *testing.T) {
	cf := NewConfiguration()
	//GetProfile
	p, ok := cf.GetProfile("hh")
	assert.False(t, ok)
	assert.Equal(t, Profile{Name: "hh"}, p)
	p, _ = cf.GetProfile("default")
	assert.Equal(t, Profile{Name: "default", Mode: "", OutputFormat: "json", Language: "en"}, p)

	//PutProfile
	assert.Len(t, cf.Profiles, 1)
	cf.PutProfile(Profile{Name: "test", Mode: AK, OutputFormat: "json", Language: "en"})
	assert.Len(t, cf.Profiles, 2)
	assert.Equal(t, Profile{Name: "test", Mode: AK, OutputFormat: "json", Language: "en"}, cf.Profiles[1])
	cf.PutProfile(Profile{Name: "test", Mode: StsToken, OutputFormat: "json", Language: "en"})
	assert.Len(t, cf.Profiles, 2)
	assert.Equal(t, Profile{Name: "test", Mode: StsToken, OutputFormat: "json", Language: "en"}, cf.Profiles[1])

	//GetCurrentProfile
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)
	AddFlags(ctx.Flags())

	os.Setenv("ACCESS_KEY_ID", "")
	os.Setenv("ACCESS_KEY_SECRET", "")
	os.Setenv("ALIBABACLOUD_PROFILE", "test")
	p = cf.GetCurrentProfile(ctx)
	assert.Equal(t, Profile{Name: "test", Mode: StsToken, OutputFormat: "json", Language: "en"}, p)
	cf.PutProfile(Profile{Name: "test2", Mode: StsToken, OutputFormat: "json", Language: "en"})
	cf.CurrentProfile = "test2"
	p = cf.GetCurrentProfile(ctx)
	assert.Equal(t, Profile{Name: "test2", Mode: StsToken, OutputFormat: "json", Language: "en"}, p)
	cf.CurrentProfile = "default"
	p = cf.GetCurrentProfile(ctx)
	assert.Equal(t, Profile{Name: "test", Mode: StsToken, OutputFormat: "json", Language: "en"}, p)
	os.Setenv("ALIBABA_CLOUD_PROFILE", "test2")
	p = cf.GetCurrentProfile(ctx)
	assert.Equal(t, Profile{Name: "test", Mode: StsToken, OutputFormat: "json", Language: "en"}, p)
	os.Setenv("ALIBABACLOUD_PROFILE", "")
	p = cf.GetCurrentProfile(ctx)
	assert.Equal(t, Profile{Name: "test2", Mode: StsToken, OutputFormat: "json", Language: "en"}, p)
	os.Setenv("ALICLOUD_PROFILE", "test")
	p = cf.GetCurrentProfile(ctx)
	assert.Equal(t, Profile{Name: "test2", Mode: StsToken, OutputFormat: "json", Language: "en"}, p)
	os.Setenv("ALIBABA_CLOUD_PROFILE", "")
	p = cf.GetCurrentProfile(ctx)
	assert.Equal(t, Profile{Name: "test", Mode: StsToken, OutputFormat: "json", Language: "en"}, p)
	os.Setenv("ALICLOUD_PROFILE", "")
	p = cf.GetCurrentProfile(ctx)
	assert.Equal(t, Profile{Name: "default", Mode: "", OutputFormat: "json", Language: "en"}, p)
}

func TestLoadProfile(t *testing.T) {
	originhook := hookLoadOrCreateConfiguration
	w := new(bytes.Buffer)
	defer func() {
		hookLoadOrCreateConfiguration = originhook
	}()
	hookLoadOrCreateConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
		return func(path string) (*Configuration, error) {
			return &Configuration{CurrentProfile: "default", Profiles: []Profile{{Name: "default", Mode: AK, AccessKeyId: "default_aliyun_access_key_id", AccessKeySecret: "default_aliyun_access_key_secret", OutputFormat: "json"}, {Name: "aaa", Mode: AK, AccessKeyId: "sdf", AccessKeySecret: "ddf", OutputFormat: "json"}}}, nil
		}
	}
	//testcase 1
	p, err := LoadProfile(GetConfigPath()+"/"+configFile, "")
	assert.Nil(t, err)
	p.parent = nil
	assert.Equal(t, Profile{Name: "default", Mode: AK, AccessKeyId: "default_aliyun_access_key_id", AccessKeySecret: "default_aliyun_access_key_secret", OutputFormat: "json"}, p)

	//testcase 2
	_, err = LoadProfile(GetConfigPath()+"/"+configFile, "hello")
	assert.EqualError(t, err, "unknown profile hello, run configure to check")

	//LoadCurrentProfile testcase
	w.Reset()
	p, err = LoadOrCreateDefaultProfile()
	assert.Nil(t, err)
	p.parent = nil
	assert.Equal(t, Profile{Name: "default", Mode: AK, AccessKeyId: "default_aliyun_access_key_id", AccessKeySecret: "default_aliyun_access_key_secret", OutputFormat: "json"}, p)

	//testcase 3
	hookLoadOrCreateConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
		return func(path string) (*Configuration, error) {
			return &Configuration{}, errors.New("error")
		}
	}
	w.Reset()
	p, err = LoadProfile(GetConfigPath()+"/"+configFile, "")
	assert.Empty(t, p)
	assert.EqualError(t, err, "init config failed error")
}

func TestHomePath(t *testing.T) {
	if runtime.GOOS == "windows" {
		assert.Equal(t, os.Getenv("USERPROFILE"), GetHomePath())
	} else {
		assert.Equal(t, os.Getenv("HOME"), GetHomePath())
	}
}

func TestGetConfigPath(t *testing.T) {
	orighookGetHomePath := hookGetHomePath
	defer func() {
		os.RemoveAll("./.aliyun")
		hookGetHomePath = orighookGetHomePath
	}()
	hookGetHomePath = func(fn func() string) func() string {
		return func() string {
			return "."
		}
	}
	assert.Equal(t, "./.aliyun", GetConfigPath())
}

func TestNewConfigFromBytes(t *testing.T) {
	bytesConf := `{
		"current": "",
		"profiles": [
			{
				"name": "default",
				"mode": "AK",
				"access_key_id": "access_key_id",
				"access_key_secret": "access_key_secret",
				"sts_token": "",
				"ram_role_name": "",
				"ram_role_arn": "",
				"ram_session_name": "",
				"private_key": "",
				"key_pair_name": "",
				"expired_seconds": 0,
				"verified": "",
				"region_id": "cn-hangzhou",
				"output_format": "json",
				"language": "en",
				"site": "",
				"read_timeout": 0,
				"retry_count": 0
			}
		],
		"meta_path": ""
	}`

	conf, err := NewConfigFromBytes([]byte(bytesConf))
	assert.Nil(t, err)
	assert.Equal(t, &Configuration{Profiles: []Profile{{Language: "en", Name: "default", Mode: "AK", AccessKeyId: "access_key_id", AccessKeySecret: "access_key_secret", RegionId: "cn-hangzhou", OutputFormat: "json"}}}, conf)
}

func TestSaveConfiguration(t *testing.T) {
	orighookGetHomePath := hookGetHomePath
	defer func() {
		os.RemoveAll("./.aliyun")
		hookGetHomePath = orighookGetHomePath
	}()
	hookGetHomePath = func(fn func() string) func() string {
		return func() string {
			return "."
		}
	}
	conf := &Configuration{Profiles: []Profile{{Language: "en", Name: "default", Mode: "AK", AccessKeyId: "access_key_id", AccessKeySecret: "access_key_secret", RegionId: "cn-hangzhou", OutputFormat: "json"}}}
	bytes, err := json.MarshalIndent(conf, "", "\t")
	assert.Nil(t, err)
	err = SaveConfiguration(conf)
	assert.Nil(t, err)
	file, err := os.Open(filepath.Join(GetConfigPath(), configFile))
	assert.Nil(t, err)
	buf := make([]byte, 1024)
	n, _ := file.Read(buf)
	file.Close()
	assert.Equal(t, string(bytes), string(buf[:n]))
}

func TestAtomicWriteFile_OverwriteExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	err := os.WriteFile(path, []byte(`{"current":"old"}`), 0600)
	assert.NoError(t, err)

	newContent := []byte(`{"current":"new","profiles":[]}`)
	err = atomicWriteFile(path, newContent, 0600)
	assert.NoError(t, err)

	got, err := os.ReadFile(path)
	assert.NoError(t, err)
	assert.Equal(t, string(newContent), string(got))

	entries, err := os.ReadDir(dir)
	assert.NoError(t, err)
	for _, e := range entries {
		assert.False(t, strings.Contains(e.Name(), ".tmp-"), "temp file should be cleaned: %s", e.Name())
	}
}

func TestAtomicWriteFile_RenameFailurePreservesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	oldContent := []byte(`{"current":"old"}`)
	assert.NoError(t, os.WriteFile(path, oldContent, 0600))

	rename := func(oldPath, newPath string) error {
		return errors.New("injected rename failure")
	}

	err := atomicWriteFileWithRename(path, []byte(`{"current":"new"}`), 0600, rename)
	assert.ErrorContains(t, err, "injected rename failure")

	got, err := os.ReadFile(path)
	assert.NoError(t, err)
	assert.Equal(t, oldContent, got)

	entries, err := os.ReadDir(dir)
	assert.NoError(t, err)
	for _, entry := range entries {
		assert.False(t, strings.Contains(entry.Name(), ".tmp-"), "temp file should be cleaned: %s", entry.Name())
	}
}

func TestAtomicWriteFile_PreservesSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks may require elevated privileges on Windows")
	}

	targetDir := t.TempDir()
	targetPath := filepath.Join(targetDir, "config.json")
	assert.NoError(t, os.WriteFile(targetPath, []byte(`{"current":"old"}`), 0600))

	linkDir := t.TempDir()
	linkPath := filepath.Join(linkDir, "config.json")
	assert.NoError(t, os.Symlink(targetPath, linkPath))

	newContent := []byte(`{"current":"new"}`)
	assert.NoError(t, atomicWriteFile(linkPath, newContent, 0600))

	info, err := os.Lstat(linkPath)
	assert.NoError(t, err)
	assert.NotZero(t, info.Mode()&os.ModeSymlink)
	got, err := os.ReadFile(targetPath)
	assert.NoError(t, err)
	assert.Equal(t, newContent, got)
}

func TestAtomicWriteFile_DanglingSymlinkFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks may require elevated privileges on Windows")
	}

	dir := t.TempDir()
	linkPath := filepath.Join(dir, "config.json")
	assert.NoError(t, os.Symlink(filepath.Join(dir, "missing.json"), linkPath))

	err := atomicWriteFile(linkPath, []byte(`{"current":"new"}`), 0600)
	assert.ErrorContains(t, err, "failed to resolve config symlink")
}

func TestAtomicWriteFile_CreateTempFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "config.json")

	err := atomicWriteFile(path, []byte(`{"current":"new"}`), 0600)
	assert.ErrorContains(t, err, "failed to create temp config")
}

func TestAtomicWriteFile_InspectPathFailure(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	assert.NoError(t, os.WriteFile(blocker, []byte("not-a-dir"), 0600))
	path := filepath.Join(blocker, "config.json")

	err := atomicWriteFile(path, []byte(`{"current":"new"}`), 0600)
	assert.ErrorContains(t, err, "failed to inspect config path")
}

func TestAtomicWriteFile_TempFileFailures(t *testing.T) {
	originCreateTemp := createAtomicTempFile
	defer func() {
		createAtomicTempFile = originCreateTemp
	}()

	tests := []struct {
		name    string
		file    *fakeAtomicTempFile
		wantErr string
	}{
		{
			name:    "chmod",
			file:    &fakeAtomicTempFile{chmodErr: errors.New("chmod failed")},
			wantErr: "chmod failed",
		},
		{
			name:    "write",
			file:    &fakeAtomicTempFile{writeErr: errors.New("write failed")},
			wantErr: "write failed",
		},
		{
			name:    "short write",
			file:    &fakeAtomicTempFile{writeN: 1},
			wantErr: "short write",
		},
		{
			name:    "sync",
			file:    &fakeAtomicTempFile{syncErr: errors.New("sync failed")},
			wantErr: "sync failed",
		},
		{
			name:    "close",
			file:    &fakeAtomicTempFile{closeErr: errors.New("close failed")},
			wantErr: "close failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempPath := filepath.Join(t.TempDir(), "config.json.tmp")
			tt.file.name = tempPath
			createAtomicTempFile = func(dir, pattern string) (atomicTempFile, error) {
				return tt.file, nil
			}

			err := atomicWriteFile(filepath.Join(t.TempDir(), "config.json"), []byte(`{"current":"new"}`), 0600)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestSaveConfiguration_OverwriteExisting(t *testing.T) {
	orighookGetHomePath := hookGetHomePath
	defer func() {
		os.RemoveAll("./.aliyun")
		hookGetHomePath = orighookGetHomePath
	}()
	hookGetHomePath = func(fn func() string) func() string {
		return func() string {
			return "."
		}
	}

	oldConf := &Configuration{
		CurrentProfile: "old",
		Profiles:       []Profile{{Language: "en", Name: "old", Mode: "AK", AccessKeyId: "old_id", AccessKeySecret: "old_secret", RegionId: "cn-hangzhou", OutputFormat: "json"}},
	}
	assert.NoError(t, SaveConfiguration(oldConf))

	newConf := &Configuration{
		CurrentProfile: "default",
		Profiles:       []Profile{{Language: "en", Name: "default", Mode: "AK", AccessKeyId: "new_id", AccessKeySecret: "new_secret", RegionId: "cn-beijing", OutputFormat: "json"}},
	}
	assert.NoError(t, SaveConfiguration(newConf))

	path := filepath.Join(GetConfigPath(), configFile)
	loaded, err := LoadConfigurationFromFile(path)
	assert.NoError(t, err)
	assert.Equal(t, "default", loaded.CurrentProfile)
	assert.Equal(t, "new_id", loaded.Profiles[0].AccessKeyId)
	assert.Equal(t, "cn-beijing", loaded.Profiles[0].RegionId)

	entries, err := os.ReadDir(GetConfigPath())
	assert.NoError(t, err)
	for _, e := range entries {
		assert.False(t, strings.Contains(e.Name(), ".tmp-"), "temp file should be cleaned: %s", e.Name())
	}
}

func TestSaveConfigurationWithContext_CustomPathCreatesDir(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)
	AddFlags(ctx.Flags())
	customPath := filepath.Join(t.TempDir(), "nested", "config.json")
	ConfigurePathFlag(ctx.Flags()).SetAssigned(true)
	ConfigurePathFlag(ctx.Flags()).SetValue(customPath)

	conf := &Configuration{
		CurrentProfile: "default",
		Profiles:       []Profile{{Language: "en", Name: "default", Mode: "AK", AccessKeyId: "new_id", AccessKeySecret: "new_secret", RegionId: "cn-beijing", OutputFormat: "json"}},
	}
	assert.NoError(t, SaveConfigurationWithContext(ctx, conf))

	loaded, err := LoadConfigurationFromFile(customPath)
	assert.NoError(t, err)
	assert.Equal(t, "default", loaded.CurrentProfile)
	assert.Equal(t, "new_id", loaded.Profiles[0].AccessKeyId)
}

func TestLoadOrCreateConfiguration(t *testing.T) {
	orighookGetHomePath := hookGetHomePath
	defer func() {
		os.RemoveAll("./.aliyun")
		hookGetHomePath = orighookGetHomePath
	}()
	hookGetHomePath = func(fn func() string) func() string {
		return func() string {
			return "."
		}
	}
	w := new(bytes.Buffer)

	//testcase 1
	cf, err := LoadOrCreateConfiguration(GetConfigPath() + "/" + configFile)
	assert.Nil(t, err)
	assert.Equal(t, &Configuration{CurrentProfile: "default", Profiles: []Profile{{Name: "default", Mode: "", OutputFormat: "json", Language: "en"}}}, cf)
	conf := &Configuration{Profiles: []Profile{{Language: "en", Name: "default", Mode: "AK", AccessKeyId: "access_key_id", AccessKeySecret: "access_key_secret", RegionId: "cn-hangzhou", OutputFormat: "json"}}}
	err = SaveConfiguration(conf)
	assert.Nil(t, err)

	//testcase 2
	w.Reset()
	cf, err = LoadOrCreateConfiguration(GetConfigPath() + "/" + configFile)
	assert.Equal(t, &Configuration{CurrentProfile: "", Profiles: []Profile{{Name: "default", Mode: "AK", AccessKeyId: "access_key_id", AccessKeySecret: "access_key_secret", RegionId: "cn-hangzhou", OutputFormat: "json", Language: "en"}}}, cf)
	assert.Nil(t, err)

}

func TestLoadProfileWithContext(t *testing.T) {
	originhook := hookLoadOrCreateConfiguration
	defer func() {
		hookLoadOrCreateConfiguration = originhook
	}()
	hookLoadOrCreateConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
		return func(path string) (*Configuration, error) {
			return &Configuration{CurrentProfile: "default", Profiles: []Profile{{Name: "default", Mode: AK, AccessKeyId: "default_aliyun_access_key_id", AccessKeySecret: "default_aliyun_access_key_secret", OutputFormat: "json"}, {Name: "aaa", Mode: AK, AccessKeyId: "sdf", AccessKeySecret: "ddf", OutputFormat: "json"}}}, nil
		}
	}
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)
	AddFlags(ctx.Flags())

	//testcase 1
	_, err := LoadProfileWithContext(ctx)
	assert.EqualError(t, err, "region can't be empty")

	//testcase 2
	ctx.Flags().Get("profile").SetAssigned(true)
	_, err = LoadProfileWithContext(ctx)
	assert.EqualError(t, err, "region can't be empty")
}

func TestLoadProfileWithContextWhenIGNORE_PROFILE(t *testing.T) {
	os.Setenv("ALIBABA_CLOUD_IGNORE_PROFILE", "TRUE")
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)
	ctx.SetInConfigureMode(true)
	AddFlags(ctx.Flags())
	ctx.Flags().Get("access-key-id").SetAssigned(true)
	ctx.Flags().Get("access-key-id").SetValue("test-ak-id")
	ctx.Flags().Get("access-key-secret").SetAssigned(true)
	ctx.Flags().Get("access-key-secret").SetValue("test-ak-secret")
	p, err := LoadProfileWithContext(ctx)
	assert.Nil(t, err)
	assert.Equal(t, "default", p.Name)
	assert.Equal(t, "cn-hangzhou", p.RegionId)
	assert.Equal(t, AK, p.Mode)
	// reset
	os.Setenv("ALIBABA_CLOUD_IGNORE_PROFILE", "")
}

func TestLoadProfileWithContext_Anonymous(t *testing.T) {
	// C-01: 无 config.json 也能走匿名模式
	t.Run("C-01: Anonymous flag without config.json", func(t *testing.T) {
		stdout := new(bytes.Buffer)
		stderr := new(bytes.Buffer)
		ctx := cli.NewCommandContext(stdout, stderr)
		AddFlags(ctx.Flags())
		ModeFlag(ctx.Flags()).SetAssigned(true)
		ModeFlag(ctx.Flags()).SetValue("Anonymous")
		ctx.Flags().Get("region").SetAssigned(true)
		ctx.Flags().Get("region").SetValue("cn-hangzhou")
		ctx.SetInConfigureMode(true)
		p, err := LoadProfileWithContext(ctx)
		assert.Nil(t, err)
		assert.Equal(t, Anonymous, p.Mode)
		assert.Equal(t, "cn-hangzhou", p.RegionId)
		assert.Equal(t, "Anonymous", p.OpenAPIAuthType())
		cred, credErr := p.GetCredential(ctx, nil)
		assert.NoError(t, credErr)
		assert.Nil(t, cred)
	})

	// C-03: --region 覆盖默认 cn-hangzhou
	t.Run("C-03: --region overrides default region", func(t *testing.T) {
		stdout := new(bytes.Buffer)
		stderr := new(bytes.Buffer)
		ctx := cli.NewCommandContext(stdout, stderr)
		AddFlags(ctx.Flags())
		ModeFlag(ctx.Flags()).SetAssigned(true)
		ModeFlag(ctx.Flags()).SetValue("Anonymous")
		ctx.Flags().Get("region").SetAssigned(true)
		ctx.Flags().Get("region").SetValue("us-east-1")
		ctx.SetInConfigureMode(true)
		p, err := LoadProfileWithContext(ctx)
		assert.Nil(t, err)
		assert.Equal(t, Anonymous, p.Mode)
		assert.Equal(t, "us-east-1", p.RegionId)
	})

	// C-03 extra: env-based ALIBABA_CLOUD_PROFILE_MODE + --region
	t.Run("C-03 extra: env Anonymous with --region", func(t *testing.T) {
		t.Setenv("ALIBABA_CLOUD_PROFILE_MODE", "anonymous")
		stdout := new(bytes.Buffer)
		stderr := new(bytes.Buffer)
		ctx := cli.NewCommandContext(stdout, stderr)
		AddFlags(ctx.Flags())
		ctx.Flags().Get("region").SetAssigned(true)
		ctx.Flags().Get("region").SetValue("ap-southeast-1")
		ctx.SetInConfigureMode(true)
		p, err := LoadProfileWithContext(ctx)
		assert.Nil(t, err)
		assert.Equal(t, Anonymous, p.Mode)
		assert.Equal(t, "ap-southeast-1", p.RegionId)
	})
}

func TestGetHomePath(t *testing.T) {
	home := GetHomePath()
	assert.NotEqual(t, "", home)
}

func TestGetProfileName(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)
	AddFlags(ctx.Flags())
	// default case: no flag, no env
	name := getProfileName(ctx)
	assert.Equal(t, name, "")

	// case 1: with flag
	ctx.Flags().Get("profile").SetAssigned(true)
	ctx.Flags().Get("profile").SetValue("FromProfileFlag")
	name = getProfileName(ctx)
	assert.Equal(t, name, "FromProfileFlag")

	// case 2: with env
	ctx.Flags().Get("profile").SetAssigned(false)
	ctx.Flags().Get("profile").SetValue("")
	name = getProfileName(ctx)
	assert.Equal(t, name, "") // reset flag
	os.Setenv("ALIBABA_CLOUD_PROFILE", "profileName")
	name = getProfileName(ctx)
	assert.Equal(t, name, "profileName")
	os.Setenv("ALIBABA_CLOUD_PROFILE", "") // reset env
}

func TestGetConfigurePath(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)
	AddFlags(ctx.Flags())
	// default case: no flag, no env
	p := getConfigurePath(ctx)
	assert.Contains(t, p, ".aliyun/config.json")

	// case 1: with flag
	ctx.Flags().Get("config-path").SetAssigned(true)
	ctx.Flags().Get("config-path").SetValue("/path/to/config.json")
	p = getConfigurePath(ctx)
	assert.Equal(t, p, "/path/to/config.json")
}

type fakeAtomicTempFile struct {
	name     string
	writeN   int
	chmodErr error
	writeErr error
	syncErr  error
	closeErr error
}

func (f *fakeAtomicTempFile) Name() string {
	return f.name
}

func (f *fakeAtomicTempFile) Chmod(os.FileMode) error {
	return f.chmodErr
}

func (f *fakeAtomicTempFile) Write(data []byte) (int, error) {
	if f.writeErr != nil {
		return 0, f.writeErr
	}
	if f.writeN != 0 {
		return f.writeN, nil
	}
	return len(data), nil
}

func (f *fakeAtomicTempFile) Sync() error {
	return f.syncErr
}

func (f *fakeAtomicTempFile) Close() error {
	return f.closeErr
}
