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
	"regexp"
	"strings"

	"github.com/aliyun/aliyun-cli/v3/util"
)

var (
	pathUsersPattern   = regexp.MustCompile(`/Users/[^/\s]+/`)
	pathHomePattern    = regexp.MustCompile(`/home/[^/\s]+/`)
	pathWinPattern     = regexp.MustCompile(`(?i)[A-Z]:\\Users\\[^\\]+\\`)
	credentialPatterns = []*regexp.Regexp{
		regexp.MustCompile(`LTAI[A-Za-z0-9]{12,}`),
		regexp.MustCompile(`(?i)AccessKey(?:Id|Secret)?`),
		regexp.MustCompile(`(?i)Authorization`),
		regexp.MustCompile(`(?i)Bearer\s+\S+`),
		regexp.MustCompile(`(?i)[0-9a-f]{32,}`),
	}
)

func SanitizeSummary(raw string) string {
	if raw == "" {
		return ""
	}
	s := util.SanitizeUserAgent(raw)
	s = pathUsersPattern.ReplaceAllString(s, "/Users/<REDACTED>/")
	s = pathHomePattern.ReplaceAllString(s, "/home/<REDACTED>/")
	s = pathWinPattern.ReplaceAllString(s, `C:\Users\<REDACTED>\`)
	for _, re := range credentialPatterns {
		s = re.ReplaceAllString(s, "<REDACTED>")
	}
	if idx := strings.Index(s, "?"); idx >= 0 {
		s = s[:idx] + "?<REDACTED>"
	}
	s = strings.NewReplacer("'", "_", "\"", "_", "\r\n", " ", "\n", " ", "\r", " ").Replace(s)
	s = strings.TrimSpace(s)
	if len(s) > 200 {
		s = s[:200]
	}
	return s
}
