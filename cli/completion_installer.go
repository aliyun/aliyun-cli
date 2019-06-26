// Copyright 1999-2019 Alibaba Group Holding Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package cli

import (
	"fmt"
	"io"

	"github.com/aliyun/aliyun-cli/i18n"
)

var hookGetBinaryPath = func(fn func() (string, error)) func() (string, error) {
	return fn
}
var uninstallFlag = &Flag{
	Name:  "uninstall",
	Short: i18n.T("uninstall auto completion", "卸载自动完成"),
}

func NewAutoCompleteCommand() *Command {
	cmd := &Command{
		Name: "auto-completion",
		Short: i18n.T(
			"enable auto completion",
			"启用自动完成"),
		Usage: "auto-completion [--uninstall]",
		Run: func(ctx *Context, args []string) error {
			//s, _ := os.Executable()
			//fmt.Printf("%s \n", s)
			//
			//if f := rcFile(".zshrc"); f != "" {
			//	// i = append(i, zshInstaller{f})
			//	fmt.Printf("zshInstaller: %s\n", f)
			//}
			if uninstallFlag.IsAssigned() {
				uninstallCompletion(ctx.Writer(), "aliyun")
			} else {
				installCompletion(ctx.Writer(), "aliyun")
			}
			return nil
		},
	}
	cmd.Flags().Add(uninstallFlag)
	return cmd
}

func installCompletion(w io.Writer, cmd string) {
	bin, err := getBinaryPath()
	if err != nil {
		Errorf(w, "can't get binary path %s", err)
		return
	}

	for _, i := range completionInstallers() {
		err := i.Install(cmd, bin)
		if err != nil {
			Errorf(w, "install completion failed for %s %s\n", bin, err)
		}
	}
}

func uninstallCompletion(w io.Writer, cmd string) {
	bin, err := hookGetBinaryPath(getBinaryPath)()
	if err != nil {
		Errorf(w, "can't get binary path %s", err)
		return
	}

	for _, i := range completionInstallers() {
		err := i.Uninstall(cmd, bin)
		if err != nil {
			Errorf(w, "uninstall %s failed\n", err)
		}
	}
}

func completionInstallers() (i []completionInstaller) {
	for _, rc := range [...]string{".bashrc", ".bash_profile", ".bash_login", ".profile"} {
		if f := rcFile(rc); f != "" {
			i = append(i, bashInstaller{f})
			break
		}
	}
	if f := rcFile(".zshrc"); f != "" {
		i = append(i, zshInstaller{f})
	}
	return
}

type completionInstaller interface {
	GetName() string
	Install(cmd string, bin string) error
	Uninstall(cmd string, bin string) error
}

// (un)install in zshInstaller
// basically adds/remove from .zshrc:
//
// autoload -U +X bashcompinit && bashcompinit"
// complete -C </path/to/completion/command> <command>
type zshInstaller struct {
	rc string
}

func (z zshInstaller) GetName() string {
	return "zsh"
}

func (z zshInstaller) Install(cmd, bin string) error {
	completeCmd := z.cmd(cmd, bin)
	if lineInFile(z.rc, completeCmd) {
		return fmt.Errorf("already installed in %s", z.rc)
	}

	bashCompInit := "autoload -U +X bashcompinit && bashcompinit"
	if !lineInFile(z.rc, bashCompInit) {
		completeCmd = bashCompInit + "\n" + completeCmd
	}

	return appendToFile(z.rc, completeCmd)
}

func (z zshInstaller) Uninstall(cmd, bin string) error {
	completeCmd := z.cmd(cmd, bin)
	if !lineInFile(z.rc, completeCmd) {
		return fmt.Errorf("does not installed in %s", z.rc)
	}

	return removeFromFile(z.rc, completeCmd)
}

func (zshInstaller) cmd(cmd, bin string) string {
	return fmt.Sprintf("complete -o nospace -F %s %s", bin, cmd)
}

// (un)install in bashInstaller
// basically adds/remove from .bashrc:
//
// complete -C </path/to/completion/command> <command>
type bashInstaller struct {
	rc string
}

func (b bashInstaller) GetName() string {
	return "bash"
}

func (b bashInstaller) Install(cmd, bin string) error {
	completeCmd := b.cmd(cmd, bin)
	if lineInFile(b.rc, completeCmd) {
		return fmt.Errorf("already installed in %s", b.rc)
	}
	return appendToFile(b.rc, completeCmd)
}

func (b bashInstaller) Uninstall(cmd, bin string) error {
	completeCmd := b.cmd(cmd, bin)
	if !lineInFile(b.rc, completeCmd) {
		return fmt.Errorf("does not installed in %s", b.rc)
	}

	return removeFromFile(b.rc, completeCmd)
}

func (bashInstaller) cmd(cmd, bin string) string {
	return fmt.Sprintf("complete -C %s %s", bin, cmd)
}
