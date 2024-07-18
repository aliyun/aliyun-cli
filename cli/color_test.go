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
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestColor(t *testing.T) {
	assert.False(t, isNoColor())
	os.Setenv("NO_COLOR", "1")
	assert.True(t, isNoColor())
	os.Setenv("NO_COLOR", "true")
	assert.True(t, isNoColor())
	os.Setenv("NO_COLOR", "")
	assert.False(t, isNoColor())
}

func TestColorized(t *testing.T) {
	assert.Equal(t, "\x1b[0;31mtext\x1b[0m", Colorized(Red, "text"))
	os.Setenv("NO_COLOR", "1")
	assert.Equal(t, "text", Colorized(Red, "text"))
	os.Setenv("NO_COLOR", "")
}

func TestCotainWriter(t *testing.T) {
	writer := new(bytes.Buffer)
	n, err := PrintWithColor(writer, "red", "a")
	assert.Equal(t, "reda\033[0m", writer.String())
	assert.Equal(t, 8, n)
	assert.Nil(t, err)

	writer.Reset()
	n, err = Notice(writer, "a")
	assert.Equal(t, "\033[1;33ma\x1b[0m", writer.String())
	assert.Equal(t, 12, n)
	assert.Nil(t, err)

	writer.Reset()
	n, err = Error(writer, "a")
	assert.Equal(t, "\033[1;31ma\x1b[0m", writer.String())
	assert.Equal(t, 12, n)
	assert.Nil(t, err)

	writer.Reset()
	n, err = Noticef(writer, "%s", "a")
	assert.Equal(t, "\033[1;33ma\x1b[0m", writer.String())
	assert.Equal(t, 12, n)
	assert.Nil(t, err)

	writer.Reset()
	n, err = Errorf(writer, "%s", "a")
	assert.Equal(t, "\033[1;31ma\x1b[0m", writer.String())
	assert.Equal(t, 12, n)
	assert.Nil(t, err)

	writer.Reset()
	n, err = PrintfWithColor(writer, "red", "%s", "a")
	assert.Equal(t, "reda\033[0m", writer.String())
	assert.Equal(t, 8, n)
	assert.Nil(t, err)
}
