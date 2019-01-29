/*
 * Copyright (C) 1999-2019 Alibaba Group Holding Limited
 */
package cli

import (
	"bytes"
	"testing"

	"github.com/aliyun/aliyun-cli/i18n"
	"github.com/stretchr/testify/assert"
)

func TestGetVersion(t *testing.T) {
	assert.Equal(t, Version, GetVersion())
}

func TestNewVersionCommand(t *testing.T) {
	excmd := &Command{
		Name:   "version",
		Short:  i18n.T("print current version", "打印当前版本号"),
		Hidden: true,
		Run: func(ctx *Context, args []string) error {
			Printf(ctx.Writer(), "%s\n", Version)
			return nil
		},
	}
	cmd := NewVersionCommand()
	assert.ObjectsAreEqualValues(excmd, cmd)

	w := new(bytes.Buffer)
	ctx := NewCommandContext(w)
	err := cmd.Run(ctx, []string{})
	assert.Nil(t, err)
	assert.Equal(t, Version+"\n", w.String())
}
