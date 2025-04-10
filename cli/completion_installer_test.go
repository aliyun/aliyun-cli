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
package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/stretchr/testify/assert"
)

func init() {
	wd, _ := os.Getwd()
	homeDir := filepath.Join(wd, "mock_user")
	os.Setenv("MOCK_USER_HOME_DIR", homeDir)
	os.Remove(homeDir)
}

func TestBashInstaller(t *testing.T) {
	var bi bashInstaller
	//GetName & cmd
	assert.Equal(t, "bash", bi.GetName())
	assert.Equal(t, "complete -C oss aliyun", bi.cmd("aliyun", "oss"))

	//Install
	err := createFile("test.txt", "ecs")
	assert.Nil(t, err)
	bi.rc = "test.txt"
	err = bi.Install("aliyun", "oss")
	assert.Nil(t, err)
	err = bi.Install("aliyun", "oss")
	assert.EqualError(t, err, "already installed in test.txt")

	//Uninstall
	err = bi.Uninstall("oss", "aliyun")
	assert.EqualError(t, err, "does not installed in test.txt")
	err = bi.Uninstall("aliyun", "oss")
	assert.Nil(t, err)
	os.Remove("test.txt")
}

func TestZshInstaller(t *testing.T) {
	var z zshInstaller
	//GetName
	assert.Equal(t, "zsh", z.GetName())
	assert.Equal(t, "complete -o nospace -F oss aliyun", z.cmd("aliyun", "oss"))

	//Install
	err := createFile("test.txt", "ecs")
	assert.Nil(t, err)
	z.rc = "test.txt"
	err = z.Install("aliyun", "oss")
	assert.Nil(t, err)
	err = z.Install("aliyun", "oss")
	assert.EqualError(t, err, "already installed in test.txt")

	//Uninstall
	assert.Nil(t, z.Uninstall("aliyun", "oss"))
	assert.EqualError(t, z.Uninstall("aliyun", "oss"), "does not installed in test.txt")
	os.Remove("test.txt")
}

func TestCompletionInstallers(t *testing.T) {
	i := completionInstallers()
	if runtime.GOOS == "windows" {
		assert.Empty(t, i)
	} else {
		assert.Nil(t, i)
	}

	path := filepath.Join(getHomeDir(), ".bashrc")
	err := createFile(path, "ecs")
	assert.Nil(t, err)
	i = completionInstallers()
	if runtime.GOOS == "windows" {
		assert.Len(t, i, 1)
	}
	path2 := filepath.Join(getHomeDir(), ".zshrc")
	err = createFile(path2, "ecs")
	assert.Nil(t, err)
	i = completionInstallers()
	assert.Len(t, i, 2)
	os.Remove(path)
	os.Remove(path2)
}

func TestCompletion(t *testing.T) {
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := NewCommandContext(w, stderr)
	installCompletion(ctx, "is this a cmd?")
	assert.Empty(t, w.String())
}

func TestNewAutoCompleteCommand(t *testing.T) {
	excmd := &Command{
		Name: "auto-completion",
		Short: i18n.T(
			"enable auto completion",
			"启用自动完成"),
		Usage: "auto-completion [--uninstall]",
		Run: func(ctx *Context, args []string) error {
			if uninstallFlag.IsAssigned() {
				uninstallCompletion(ctx, "aliyun")
			} else {
				installCompletion(ctx, "aliyun")
			}
			return nil
		},
	}
	excmd.Flags().Add(uninstallFlag)
	cmd := NewAutoCompleteCommand()
	assert.ObjectsAreEqualValues(excmd, cmd)
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := NewCommandContext(w, stderr)
	err := cmd.Run(ctx, []string{})
	assert.Nil(t, err)
	assert.Empty(t, w.String())
}

func TestUninstallCompletion(t *testing.T) {
	orighookGetBinaryPath := hookGetBinaryPath
	defer func() {
		hookGetBinaryPath = orighookGetBinaryPath
	}()
	hookGetBinaryPath = func(fn func() (string, error)) func() (string, error) {
		return func() (string, error) {
			return ".", errors.New("path error")
		}
	}

	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := NewCommandContext(w, stderr)
	uninstallCompletion(ctx, "aliyun")
	assert.Equal(t, "\x1b[1;31mcan't get binary path path error\x1b[0m", stderr.String())
}
