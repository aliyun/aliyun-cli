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

package safety

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

var stdinReader io.Reader = os.Stdin

// PromptConfirm asks the user to confirm the destructive operation.
// Returns true if user confirms (types y/yes), false otherwise.
// When stdin is not a terminal (non-interactive), returns false (fail-safe).
func PromptConfirm(w io.Writer, prompt string) bool {
	_, _ = fmt.Fprint(w, prompt)

	reader := bufio.NewReader(stdinReader)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	answer := strings.TrimSpace(strings.ToLower(line))
	return answer == "y" || answer == "yes"
}

// IsInteractive returns true if stdin is a terminal (interactive mode)
func IsInteractive() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}
