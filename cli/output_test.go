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

func TestDefaultWriter(t *testing.T) {
	w := DefaultStdoutWriter()
	stderr, ok := w.(*os.File)
	assert.True(t, ok)
	assert.ObjectsAreEqual(os.Stdout, stderr)

	buf := new(bytes.Buffer)
	n, err := Print(buf, "I am night")
	assert.Equal(t, 10, n)
	assert.Nil(t, err)

	buf.Reset()
	n, err = Println(buf, "How are you")
	assert.Equal(t, 12, n)
	assert.Nil(t, err)
	buf.Reset()
	n, err = Printf(buf, "I am %s", "fine")
	assert.Equal(t, 9, n)
	assert.Nil(t, err)
}

func TestOutput(t *testing.T) {
	w := new(bytes.Buffer)
	index, err := Print(w, "who are you")
	assert.Equal(t, 11, index)
	assert.Nil(t, err)
	assert.Equal(t, "who are you", w.String())

	w.Reset()
	index, err = Println(w, "I am MrX")
	assert.Equal(t, 9, index)
	assert.Nil(t, err)
	assert.Equal(t, "I am MrX\n", w.String())

	w.Reset()
	index, err = Printf(w, "and you%s", "?")
	assert.Equal(t, 8, index)
	assert.Nil(t, err)
	assert.Equal(t, "and you?", w.String())
}
