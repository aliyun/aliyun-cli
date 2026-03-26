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

package safety

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
)

var stdinReader io.Reader = os.Stdin

func PromptConfirm(w io.Writer, prompt string) bool {
	_, _ = fmt.Fprint(w, prompt)

	reader := bufio.NewReader(stdinReader)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	answer := strings.TrimSpace(strings.ToLower(line))
	return answer == "y" || answer == "yes"
}

func IsInteractive() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

func CheckAndConfirm(ctx *cli.Context, policy *Policy, cmd CommandInfo, skipConfirm bool) error {
	if policy == nil {
		policy = DefaultPolicy()
	}

	result := policy.Check(cmd)
	switch result.Action {
	case ActionDeny:
		return fmt.Errorf(i18n.T(
			"operation blocked by safety policy: %s %s (rule: %s)",
			"操作被安全策略拒绝: %s %s (规则: %s)",
		).GetMessage(), cmd.Product, cmd.ApiOrMethod, result.Rule.Pattern)
	case ActionConfirm:
		if skipConfirm {
			// --yes or env: treat as already confirmed
			break
		}
		if !IsInteractive() {
			return fmt.Errorf(i18n.T(
				"Safety policy requires confirmation for: %s %s\n"+
					"This operation cannot run in non-interactive mode without explicit approval. "+
					"If you are an agent, ask the user whether this operation is allowed; after they confirm (e.g. reply yes or 确认), re-run the same command with --yes.",
				"安全策略要求确认以下操作：%s %s\n"+
					"当前为非交互环境，无法自动确认。若调用方为智能体，请先向用户说明并征得同意；用户同意后（可在对话中回复 yes 或「确认」），再使用 --yes 重新执行同一命令。",
			).GetMessage(), cmd.Product, cmd.ApiOrMethod)
		}
		prompt := fmt.Sprintf(i18n.T(
			"Safety policy requires confirmation for: %s %s\nType 'yes' to proceed, anything else to cancel: ",
			"安全策略要求确认以下操作: %s %s\n输入 'yes' 继续，其他任意输入取消: ",
		).GetMessage(), cmd.Product, cmd.ApiOrMethod)
		if !PromptConfirm(ctx.Stderr(), prompt) {
			return fmt.Errorf(i18n.T(
				"operation cancelled by user",
				"操作已由用户取消",
			).GetMessage())
		}
		// Fall through - user confirmed
	case ActionAllow:
		// No restriction
	}
	return nil
}
