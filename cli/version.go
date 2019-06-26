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
	"strings"

	"github.com/aliyun/aliyun-cli/i18n"
)

//
// This variable is replaced in compile time
// `-ldflags "-X 'github.com/aliyun/aliyun-cli/cli.Version=${VERSION}'"`
var (
	Version = "0.0.1"
)

func GetVersion() string {
	return strings.Replace(Version, " ", "-", -1)
}

func NewVersionCommand() *Command {
	return &Command{
		Name:   "version",
		Short:  i18n.T("print current version", "打印当前版本号"),
		Hidden: true,
		Run: func(ctx *Context, args []string) error {
			Printf(ctx.Writer(), "%s\n", Version)
			return nil
		},
	}
}
