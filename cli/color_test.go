// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
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
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestColor(t *testing.T) {
	assert.True(t, withColor)
	DisableColor()
	assert.False(t, withColor)
	EnableColor()
	assert.True(t, withColor)
	DisableColor()
	assert.Empty(t, ProductListColor())
	SetProductListColor(Red)
	assert.Equal(t, "\x1b[0;31m", productListColor)
	assert.Empty(t, APIListColor())
	SetAPIListColor("Red")
	assert.Equal(t, "Red", apiListColor)

	assert.Equal(t, "a", colorized("", "a"))
	EnableColor()
	assert.Equal(t, "reda\033[0m", colorized("red", "a"))
}

func TestCotainWriter(t *testing.T) {
	w := new(bytes.Buffer)
	n, err := PrintWithColor(w, "red", "a")
	assert.Equal(t, "reda\033[0m", w.String())
	assert.Equal(t, 8, n)
	assert.Nil(t, err)

	w.Reset()
	n, err = Debug(w, "a")
	assert.Equal(t, "\x1b[0;37ma\x1b[0m", w.String())
	assert.Equal(t, 12, n)
	assert.Nil(t, err)

	w.Reset()
	n, err = Info(w, "a")
	assert.Equal(t, "\033[0;36ma\x1b[0m", w.String())
	assert.Equal(t, 12, n)
	assert.Nil(t, err)

	w.Reset()
	n, err = Notice(w, "a")
	assert.Equal(t, "\033[1;33ma\x1b[0m", w.String())
	assert.Equal(t, 12, n)
	assert.Nil(t, err)

	w.Reset()
	n, err = Warning(w, "a")
	assert.Equal(t, "\033[1;35ma\x1b[0m", w.String())
	assert.Equal(t, 12, n)
	assert.Nil(t, err)

	w.Reset()
	n, err = Error(w, "a")
	assert.Equal(t, "\033[1;31ma\x1b[0m", w.String())
	assert.Equal(t, 12, n)
	assert.Nil(t, err)

	w.Reset()
	n, err = Debugf(w, "%s", "a")
	assert.Equal(t, "\x1b[0;37ma\x1b[0m", w.String())
	assert.Equal(t, 12, n)
	assert.Nil(t, err)

	w.Reset()
	n, err = Infof(w, "%s", "a")
	assert.Equal(t, "\033[0;36ma\x1b[0m", w.String())
	assert.Equal(t, 12, n)
	assert.Nil(t, err)

	w.Reset()
	n, err = Noticef(w, "%s", "a")
	assert.Equal(t, "\033[1;33ma\x1b[0m", w.String())
	assert.Equal(t, 12, n)
	assert.Nil(t, err)

	w.Reset()
	n, err = Warningf(w, "%s", "a")
	assert.Equal(t, "\033[1;35ma\x1b[0m", w.String())
	assert.Equal(t, 12, n)
	assert.Nil(t, err)

	w.Reset()
	n, err = Errorf(w, "%s", "a")
	assert.Equal(t, "\033[1;31ma\x1b[0m", w.String())
	assert.Equal(t, 12, n)
	assert.Nil(t, err)

	w.Reset()
	n, err = PrintfWithColor(w, "red", "%s", "a")
	assert.Equal(t, "reda\033[0m", w.String())
	assert.Equal(t, 8, n)
	assert.Nil(t, err)
}
