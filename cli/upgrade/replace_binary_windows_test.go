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

//go:build windows

package upgrade

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReplaceBinary_Windows_NoOldFileAfterSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	current := filepath.Join(tmpDir, "aliyun.exe")
	oldSidecar := current + ".old"
	newPath := filepath.Join(tmpDir, "aliyun.new.exe")
	want := []byte("new payload")
	os.WriteFile(current, []byte("old"), 0755)
	os.WriteFile(newPath, want, 0755)

	assert.NoError(t, replaceBinary(newPath, current))

	got, err := os.ReadFile(current)
	assert.NoError(t, err)
	assert.Equal(t, want, got)

	_, err = os.Stat(oldSidecar)
	assert.True(t, os.IsNotExist(err), ".old sidecar should be removed after successful replace")
}
