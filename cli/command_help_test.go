/*
 * Copyright (C) 1999-2019 Alibaba Group Holding Limited
 */
package cli

import (
	"bytes"
	"errors"
	"testing"

	"github.com/aliyun/aliyun-cli/i18n"
	"github.com/stretchr/testify/assert"
)

func TestCmdPrint(t *testing.T) {
	w := new(bytes.Buffer)
	ctx := NewCommandContext(w)
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

	//PrintHead
	c.PrintHead(ctx)
	assert.Equal(t, "use `--profile <profileName>` to select profile\n", w.String())

	//PrintUsage
	w.Reset()
	c.PrintUsage(ctx)
	assert.Equal(t, "\nUsage:\n  aliyun [subcmd]\n", w.String())
	w.Reset()
	c.Usage = ""
	assert.Empty(t, w.String())

	//PrintSample
	w.Reset()
	c.PrintSample(ctx)
	assert.Equal(t, "\nSample:\n  aliyun oss\n", w.String())

	//PrintSubCommands
	w.Reset()
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
	c.PrintFlags(ctx)
	assert.Empty(t, w.String())
	c.flags.flags = []*Flag{&Flag{Name: "output", Shorthand: 'o'}}
	w.Reset()
	c.PrintFlags(ctx)
	assert.Equal(t, "\nFlags:\n", w.String())
	w.Reset()
	ctx.flags.flags = []*Flag{&Flag{Name: "output", Shorthand: 'o', Short: i18n.T("o test", "")}, &Flag{Name: "filter", Short: i18n.T("", "")}, &Flag{Name: "hidden", Hidden: true}}
	c.PrintFlags(ctx)
	assert.Equal(t, "\nFlags:\n  --output,-o o test\n  --filter    \n", w.String())

	//PrintFailed
	w.Reset()
	c.PrintFailed(ctx, errors.New("you failed"), "come on again")
	assert.Equal(t, "\x1b[1;31mERROR: you failed\n\x1b[0mcome on again\n", w.String())

	//PrintTail
	w.Reset()
	c.PrintTail(ctx)
	assert.Equal(t, "\nUse `aliyun --help` for more information.\n", w.String())
}
