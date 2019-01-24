/*
 * Copyright (C) 1999-2019 Alibaba Group Holding Limited
 */
package cli

import (
	"bytes"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/aliyun/aliyun-cli/i18n"
	"github.com/stretchr/testify/assert"
)

//  func TestNewAutoCompleteCommand(t *testing.T){

//  }

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
		assert.NotNil(t, i)
	}

	u, err := user.Current()
	assert.Nil(t, err)
	path := filepath.Join(u.HomeDir, ".bashrc")
	err = createFile(path, "ecs")
	assert.Nil(t, err)
	i = completionInstallers()
	if runtime.GOOS == "windows" {
		assert.Len(t, i, 1)
	}
	path2 := filepath.Join(u.HomeDir, ".zshrc")
	err = createFile(path2, "ecs")
	assert.Nil(t, err)
	i = completionInstallers()
	assert.Len(t, i, 2)
	os.Remove(path)
	os.Remove(path2)
}
func TestCompletion(t *testing.T) {
	w := new(bytes.Buffer)
	installCompletion(w, "is this a cmd?")
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
				uninstallCompletion(ctx.Writer(), "aliyun")
			} else {
				installCompletion(ctx.Writer(), "aliyun")
			}
			return nil
		},
	}
	excmd.Flags().Add(uninstallFlag)
	cmd := NewAutoCompleteCommand()
	assert.ObjectsAreEqualValues(excmd, cmd)

}
