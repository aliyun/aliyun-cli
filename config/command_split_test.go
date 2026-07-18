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

package config

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitProcessCommand(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		want    []string
		wantErr string
		// skipOnWindows: POSIX-only escape semantics
		skipOnWindows bool
	}{
		{
			name:  "simple",
			input: "cmd arg1 arg2",
			want:  []string{"cmd", "arg1", "arg2"},
		},
		{
			name:  "extra whitespace",
			input: "  cmd   arg1\targ2  ",
			want:  []string{"cmd", "arg1", "arg2"},
		},
		{
			name:  "windows path with spaces double quoted",
			input: `"C:\Program Files\tool\cred.exe" get --profile default`,
			want:  []string{`C:\Program Files\tool\cred.exe`, "get", "--profile", "default"},
		},
		{
			name:  "unix path with spaces single quoted",
			input: `'/usr/local/my tools/cred' arg`,
			want:  []string{"/usr/local/my tools/cred", "arg"},
		},
		{
			name:  "quoted argument with spaces",
			input: `tool --name "First Last"`,
			want:  []string{"tool", "--name", "First Last"},
		},
		{
			name:          "escaped space",
			input:         `tool arg\ with\ space`,
			want:          []string{"tool", "arg with space"},
			skipOnWindows: true,
		},
		{
			name:  "escaped quote inside double quotes",
			input: `tool "say \"hi\""`,
			want:  []string{"tool", `say "hi"`},
		},
		{
			name:  "empty double quoted argument",
			input: `tool "" arg`,
			want:  []string{"tool", "", "arg"},
		},
		{
			name:  "empty single quoted argument",
			input: `tool '' arg`,
			want:  []string{"tool", "", "arg"},
		},
		{
			name:  "adjacent quoted segments form one argument",
			input: `tool "a b"'c d'`,
			want:  []string{"tool", "a bc d"},
		},
		{
			name:          "backslash-newline outside quotes is line continuation",
			input:         "tool arg1 \\\n arg2",
			want:          []string{"tool", "arg1", "arg2"},
			skipOnWindows: true,
		},
		{
			name:          "backslash-newline inside double quotes is line continuation",
			input:         "tool \"a\\\nb\"",
			want:          []string{"tool", "ab"},
			skipOnWindows: true,
		},
		{
			name:    "empty",
			input:   "   ",
			wantErr: "process_command is empty",
		},
		{
			name:    "empty quoted command",
			input:   `""`,
			wantErr: "process_command is empty",
		},
		{
			name:    "unclosed double quote",
			input:   `"C:\Program Files\tool.exe`,
			wantErr: "unclosed quote",
		},
		{
			name:    "unclosed single quote",
			input:   `'/usr/bin/tool`,
			wantErr: "unclosed quote",
		},
		{
			name:          "trailing backslash",
			input:         `tool\`,
			wantErr:       "trailing backslash",
			skipOnWindows: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.skipOnWindows && runtime.GOOS == "windows" {
				t.Skip("POSIX escape semantics not used on Windows")
			}
			got, err := splitProcessCommand(tc.input)
			if tc.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestSplitProcessCommandForOS_WindowsKeepsBackslashes(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "unquoted windows path",
			input: `C:\tools\cred.exe get`,
			want:  []string{`C:\tools\cred.exe`, "get"},
		},
		{
			name:  "quoted program files path",
			input: `"C:\Program Files\tool\cred.exe" get`,
			want:  []string{`C:\Program Files\tool\cred.exe`, "get"},
		},
		{
			name:  "escaped quote still works",
			input: `tool "say \"hi\""`,
			want:  []string{"tool", `say "hi"`},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := splitProcessCommandForOS(tc.input, "windows")
			assert.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestSplitProcessCommandForOS_UnixEscapes(t *testing.T) {
	got, err := splitProcessCommandForOS(`tool arg\ with\ space`, "linux")
	assert.NoError(t, err)
	assert.Equal(t, []string{"tool", "arg with space"}, got)

	_, err = splitProcessCommandForOS(`tool\`, "linux")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "trailing backslash")
}
