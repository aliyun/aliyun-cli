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

package telemetry

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/term"
)

const (
	alibabaCloudEnvPrefix = "ALIBABA_CLOUD_"
	maxEnvironmentNames   = 64
)

var (
	safeEnvironmentNamePattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]{0,127}$`)
	safeDistroValuePattern     = regexp.MustCompile(`^[A-Za-z0-9 ._()+/-]{1,64}$`)
	safeLocalePattern          = regexp.MustCompile(`^[A-Za-z0-9_.@-]{1,64}$`)
)

var stdinIsTerminal = term.IsTerminal

var knownShellTypes = map[string]string{
	"ash":        "ash",
	"bash":       "bash",
	"cmd":        "cmd",
	"csh":        "csh",
	"dash":       "dash",
	"elvish":     "elvish",
	"fish":       "fish",
	"ksh":        "ksh",
	"nu":         "nu",
	"powershell": "powershell",
	"pwsh":       "pwsh",
	"sh":         "sh",
	"tcsh":       "tcsh",
	"xonsh":      "xonsh",
	"zsh":        "zsh",
}

func alibabaCloudEnvironmentNames() []string {
	names := make([]string, 0)
	for _, entry := range os.Environ() {
		name, _, ok := strings.Cut(entry, "=")
		if !ok ||
			!strings.HasPrefix(name, alibabaCloudEnvPrefix) ||
			!safeEnvironmentNamePattern.MatchString(name) {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) > maxEnvironmentNames {
		names = names[:maxEnvironmentNames]
	}
	return names
}

func detectShellType() string {
	value := strings.TrimSpace(os.Getenv("SHELL"))
	if value == "" {
		value = strings.TrimSpace(os.Getenv("COMSPEC"))
	}
	name := strings.ToLower(filepath.Base(value))
	name = strings.TrimSuffix(name, ".exe")
	if shellType, ok := knownShellTypes[name]; ok {
		return shellType
	}
	if name != "" {
		return "other"
	}
	return ""
}

func detectStdinTTY() bool {
	return stdinIsTerminal(int(os.Stdin.Fd()))
}

func detectLocale() string {
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		value := strings.TrimSpace(os.Getenv(key))
		if safeLocalePattern.MatchString(value) {
			return value
		}
	}
	return ""
}

func detectLinuxDistro() (name, id, version string) {
	if runtime.GOOS != "linux" {
		return "", "", ""
	}
	return readLinuxDistro("/etc/os-release", "/usr/lib/os-release")
}

func readLinuxDistro(paths ...string) (name, id, version string) {
	for _, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			continue
		}
		name, id, version = parseOSRelease(file)
		_ = file.Close()
		return name, id, version
	}
	return "", "", ""
}

func parseOSRelease(file *os.File) (name, id, version string) {
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		key, value, ok := strings.Cut(scanner.Text(), "=")
		if !ok {
			continue
		}
		value = decodeOSReleaseValue(strings.TrimSpace(value))
		value = sanitizeDistroValue(value)
		switch strings.TrimSpace(key) {
		case "NAME":
			name = value
		case "ID":
			id = value
		case "VERSION_ID":
			version = value
		}
	}
	return name, id, version
}

func decodeOSReleaseValue(value string) string {
	if len(value) < 2 {
		return value
	}
	if value[0] == '\'' && value[len(value)-1] == '\'' {
		return value[1 : len(value)-1]
	}
	if value[0] == '"' && value[len(value)-1] == '"' {
		if decoded, err := strconv.Unquote(value); err == nil {
			return decoded
		}
	}
	return value
}

func sanitizeDistroValue(value string) string {
	value = strings.TrimSpace(value)
	if !safeDistroValuePattern.MatchString(value) {
		return ""
	}
	return value
}
