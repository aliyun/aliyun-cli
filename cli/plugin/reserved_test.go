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

package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsReservedTopLevelCommand_Defaults(t *testing.T) {
	// The defaults hard-coded in reserved.go cover CLI-owned commands that must never be shadowed by a plugin.
	// If any of these regress to non-reserved, plugin install could silently mask a first-class command.
	for _, name := range []string{"configure", "plugin", "version", "upgrade", "oss", "ossutil", "mcp", "mock"} {
		assert.Truef(t, IsReservedTopLevelCommand(name), "expected %q to be reserved by default", name)
	}
}

func TestIsReservedTopLevelCommand_NormalizesInput(t *testing.T) {
	assert.True(t, IsReservedTopLevelCommand("Configure"))
	assert.True(t, IsReservedTopLevelCommand("  PLUGIN  "))
	assert.False(t, IsReservedTopLevelCommand(""))
	assert.False(t, IsReservedTopLevelCommand("   "))
}

func TestRegisterReservedTopLevelCommands_AppendsAndDedupes(t *testing.T) {
	// Register a fresh name and confirm it now counts as reserved.
	// The plugin package is process-global, so use a made-up name to avoid interfering with other tests.
	name := "aliyun-cli-reserved-test-name"
	assert.False(t, IsReservedTopLevelCommand(name))
	RegisterReservedTopLevelCommands([]string{name, name, "", "  "})
	assert.True(t, IsReservedTopLevelCommand(name))

	// The exported dump should be sorted and include our newly registered name exactly once.
	dump := ReservedTopLevelCommands()
	count := 0
	for _, n := range dump {
		if n == name {
			count++
		}
	}
	assert.Equal(t, 1, count, "registered name should appear exactly once")
}
