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
	"fmt"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
)

// CheckAndConfirm loads the safety policy, checks the command against it,
// and handles deny/confirm actions. Returns nil if the operation is allowed to proceed.
// When skipConfirm is true (e.g. from --yes flag), confirm actions are treated as already approved.
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
				"operation requires human confirmation but running in non-interactive mode. Aborted by safety policy.",
				"此操作需要人工确认，但当前处于非交互模式。已被安全策略中止。",
			).GetMessage())
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
