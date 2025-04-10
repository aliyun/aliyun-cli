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
	"testing"

	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/stretchr/testify/assert"
)

func TestCmdPrint(t *testing.T) {
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := NewCommandContext(w, stderr)
	c := &Command{
		Name:              "aliyun",
		EnableUnknownFlag: true,
		SuggestDistance:   2,
		Usage:             "aliyun [subcmd]",
		Short: i18n.T(
			"use `--profile <profileName>` to select profile",
			"使用 `--profile <profileName>` 指定操作的配置集",
		),
		Sample: "aliyun oss",
		flags:  NewFlagSet(),
	}

	// PrintHead
	c.PrintHead(ctx)
	assert.Equal(t, "use `--profile <profileName>` to select profile\n", w.String())

	//PrintUsage
	w.Reset()
	stderr.Reset()
	c.PrintUsage(ctx)
	assert.Equal(t, "\nUsage:\n  aliyun [subcmd]\n", w.String())
	w.Reset()
	stderr.Reset()
	c.Usage = ""
	assert.Empty(t, w.String())

	//PrintSample
	w.Reset()
	stderr.Reset()
	c.PrintSample(ctx)
	assert.Equal(t, "\nSample:\n  aliyun oss\n", w.String())

	//PrintSubCommands
	w.Reset()
	stderr.Reset()
	subcmd := &Command{
		Name:            "oss",
		SuggestDistance: 2,
		Usage:           "oss flag",
		Short: i18n.T(
			"subcmd test",
			"子命令测试",
		),
	}
	c.AddSubCommand(subcmd)
	c.PrintSubCommands(ctx)
	assert.Equal(t, "\nCommands:\n  oss   subcmd test\n", w.String())

	//PrintFlags
	w.Reset()
	stderr.Reset()
	c.PrintFlags(ctx)
	assert.Empty(t, w.String())
	c.flags.flags = []*Flag{{Name: "output", Shorthand: 'o'}}
	w.Reset()
	stderr.Reset()
	c.PrintFlags(ctx)
	assert.Equal(t, "\nFlags:\n", w.String())
	w.Reset()
	stderr.Reset()
	ctx.flags.flags = []*Flag{{Name: "output", Shorthand: 'o', Short: i18n.T("o test", "")}, {Name: "filter", Short: i18n.T("", "")}, {Name: "hidden", Hidden: true}}
	c.PrintFlags(ctx)
	assert.Equal(t, "\nFlags:\n  --output,-o o test\n  --filter    \n", w.String())

	//PrintFailed
	w.Reset()
	stderr.Reset()
	c.PrintFailed(ctx, errors.New("you failed"), "come on again")
	assert.Equal(t, "\x1b[1;31mERROR: you failed\n\x1b[0mcome on again\n", stderr.String())

	//PrintTail
	w.Reset()
	stderr.Reset()
	c.PrintTail(ctx)
	assert.Equal(t, "\nUse `aliyun --help` for more information.\n", w.String())
}
