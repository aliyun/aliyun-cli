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
	"fmt"
	"strings"
	"unicode"
)

// splitProcessCommand splits process_command into argv with quote support.
// Whitespace outside quotes separates arguments. Double/single quotes group a
// single argument so Windows paths like "C:\Program Files\tool.exe" work.
// Escape rules follow POSIX shlex: outside quotes, '\' escapes the next rune;
// inside double quotes, '\' only escapes '"', '\', '$', '`' and newline;
// inside single quotes, all characters are literal.
func splitProcessCommand(command string) ([]string, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return nil, fmt.Errorf("process_command is empty")
	}

	var args []string
	var current strings.Builder
	inSingle := false
	inDouble := false
	// hasToken tracks that a token has started even if it is empty, so quoted
	// empty arguments like `tool "" arg` keep their empty argv element.
	hasToken := false

	flush := func() {
		if hasToken {
			args = append(args, current.String())
			current.Reset()
			hasToken = false
		}
	}

	runes := []rune(command)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if inSingle {
			if r == '\'' {
				inSingle = false
			} else {
				current.WriteRune(r)
			}
			continue
		}
		if inDouble {
			if r == '"' {
				inDouble = false
				continue
			}
			if r == '\\' && i+1 < len(runes) {
				next := runes[i+1]
				if next == '"' || next == '\\' || next == '$' || next == '`' || next == '\n' {
					current.WriteRune(next)
					i++
					continue
				}
			}
			current.WriteRune(r)
			continue
		}
		if r == '\\' {
			if i+1 >= len(runes) {
				return nil, fmt.Errorf("invalid process_command: trailing backslash")
			}
			hasToken = true
			current.WriteRune(runes[i+1])
			i++
			continue
		}
		if r == '\'' {
			inSingle = true
			hasToken = true
			continue
		}
		if r == '"' {
			inDouble = true
			hasToken = true
			continue
		}
		if unicode.IsSpace(r) {
			flush()
			continue
		}
		hasToken = true
		current.WriteRune(r)
	}

	if inSingle || inDouble {
		return nil, fmt.Errorf("invalid process_command: unclosed quote")
	}
	flush()
	if len(args) == 0 || args[0] == "" {
		return nil, fmt.Errorf("process_command is empty")
	}
	return args, nil
}
