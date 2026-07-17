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

func TestSanitizeCommandAliases(t *testing.T) {
	cases := []struct {
		name    string
		command string
		raw     []string
		want    []string
	}{
		{"nil input returns nil", "hologram", nil, nil},
		{"empty entries removed", "hologram", []string{"", "  ", "hologres"}, []string{"hologres"}},
		{"self-alias stripped", "hologram", []string{"hologram", "hologres"}, []string{"hologres"}},
		{"duplicates removed case-insensitively", "hologram", []string{"hologres", "Hologres", "HOLOGRES"}, []string{"hologres"}},
		{"all invalid returns nil", "hologram", []string{"", "hologram", "HOLOGRAM"}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, sanitizeCommandAliases(tc.command, tc.raw))
		})
	}
}

func TestValidatePluginCommandAndAliases(t *testing.T) {
	// A local manifest that already has one plugin whose command / short-name / alias should trigger conflicts.
	local := &LocalManifest{
		Plugins: map[string]LocalPlugin{
			"aliyun-cli-fc": {
				Name:           "aliyun-cli-fc",
				Command:        "fc",
				CommandAliases: []string{"functions"},
			},
		},
	}

	t.Run("empty command rejected", func(t *testing.T) {
		err := validatePluginCommandAndAliases(local, "aliyun-cli-x", "", nil)
		assert.ErrorContains(t, err, "command is empty")
	})

	t.Run("reserved command rejected", func(t *testing.T) {
		err := validatePluginCommandAndAliases(local, "aliyun-cli-x", "configure", nil)
		assert.ErrorContains(t, err, "conflicts with a built-in top-level command")
	})

	t.Run("alias equal to command rejected", func(t *testing.T) {
		// sanitize would normally strip this; keep the direct call to prove the second line of defense holds.
		err := validatePluginCommandAndAliases(local, "aliyun-cli-x", "hologram", []string{"hologram"})
		assert.ErrorContains(t, err, "duplicates its command")
	})

	t.Run("duplicate alias rejected", func(t *testing.T) {
		err := validatePluginCommandAndAliases(local, "aliyun-cli-x", "hologram", []string{"hologres", "hologres"})
		assert.ErrorContains(t, err, "is duplicated")
	})

	t.Run("empty alias rejected", func(t *testing.T) {
		err := validatePluginCommandAndAliases(local, "aliyun-cli-x", "hologram", []string{""})
		assert.ErrorContains(t, err, "alias is empty")
	})

	t.Run("alias equal to reserved rejected", func(t *testing.T) {
		err := validatePluginCommandAndAliases(local, "aliyun-cli-x", "hologram", []string{"plugin"})
		assert.ErrorContains(t, err, "conflicts with a built-in top-level command")
	})

	t.Run("command conflicts with other plugin command", func(t *testing.T) {
		err := validatePluginCommandAndAliases(local, "aliyun-cli-newfc", "fc", nil)
		assert.ErrorContains(t, err, `command/alias "fc" conflicts with installed plugin "aliyun-cli-fc"`)
	})

	t.Run("alias conflicts with other plugin alias", func(t *testing.T) {
		err := validatePluginCommandAndAliases(local, "aliyun-cli-x", "hologram", []string{"functions"})
		assert.ErrorContains(t, err, `command/alias "functions" conflicts with installed plugin "aliyun-cli-fc"`)
	})

	t.Run("alias conflicts with other plugin short name", func(t *testing.T) {
		// short name of aliyun-cli-fc is "fc"; using it as alias for a new plugin must be rejected.
		err := validatePluginCommandAndAliases(local, "aliyun-cli-x", "hologram", []string{"fc"})
		assert.ErrorContains(t, err, `command/alias "fc" conflicts with installed plugin "aliyun-cli-fc"`)
	})

	t.Run("same plugin overwrite skips cross-plugin checks", func(t *testing.T) {
		// Upgrading aliyun-cli-fc to a new version: its own command/alias must not clash with itself.
		err := validatePluginCommandAndAliases(local, "aliyun-cli-fc", "fc", []string{"functions"})
		assert.NoError(t, err)
	})

	t.Run("no local plugins", func(t *testing.T) {
		err := validatePluginCommandAndAliases(nil, "aliyun-cli-x", "hologram", []string{"hologres"})
		assert.NoError(t, err)

		err = validatePluginCommandAndAliases(&LocalManifest{Plugins: map[string]LocalPlugin{}}, "aliyun-cli-x", "hologram", []string{"hologres"})
		assert.NoError(t, err)
	})
}
