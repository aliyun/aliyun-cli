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
	"testing"

	"github.com/aliyun/aliyun-cli/v3/i18n"
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
			Printf(ctx.Stdout(), "%s\n", Version)
			return nil
		},
	}
	cmd := NewVersionCommand()
	assert.ObjectsAreEqualValues(excmd, cmd)

	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := NewCommandContext(w, stderr)
	err := cmd.Run(ctx, []string{})
	assert.Nil(t, err)
	assert.Equal(t, Version+"\n", w.String())
}
