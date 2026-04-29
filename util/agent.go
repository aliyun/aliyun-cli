package util

import (
	"os"
	"strings"
)

// 设计原则：
//   - 仅输出严格 sanitize 后的 **固定枚举**，不携带工作区名、路径等可变字段；
//   - 仅依赖业界已记录在案的 **Agent 自身** 环境变量是否存在；

const (
	agentEnvProposal = "AGENT"

	agentSegmentPrefix = "agent/"

	maxAgentNameLen = 32
)

var knownAgentEnv = []struct {
	env  string
	name string
}{
	{"CURSOR_AGENT", "cursor"},
	{"CLAUDECODE", "claude-code"},
	{"CLAUDE_CODE", "claude-code"},
	{"GEMINI_CLI", "gemini-cli"},
	{"AUGMENT_AGENT", "augment"},
	{"OPENCODE", "opencode"},
	{"OPENCODE_CLIENT", "opencode"},
	{"CLINE_ACTIVE", "cline"},
	{"CODEX_SANDBOX", "codex"},
	{"QODER_AGENT", "qoder"},
	{"QODER_CLI", "qoder-cli"},
}

// Agent 标识（小写、固定枚举或 sanitize 后的 AGENT 值）；
func DetectAgentName() string {
	if v := strings.TrimSpace(os.Getenv(agentEnvProposal)); v != "" {
		if s := sanitizeAgentName(v); s != "" {
			return s
		}
	}
	for _, item := range knownAgentEnv {
		if os.Getenv(item.env) != "" {
			return item.name
		}
	}
	return ""
}

func GetAgentUserAgentSegment() string {
	name := DetectAgentName()
	if name == "" {
		return ""
	}
	return agentSegmentPrefix + name
}

const envCustomUserAgent = "ALIBABA_CLOUD_USER_AGENT"

func MergeAgentSegmentIntoPluginEnvs(envs map[string]string) {
	if envs == nil {
		return
	}
	seg := GetAgentUserAgentSegment()
	if seg == "" {
		return
	}
	base := strings.TrimSpace(envs[envCustomUserAgent])
	if base == "" {
		base = strings.TrimSpace(os.Getenv(envCustomUserAgent))
	}
	if base == "" {
		envs[envCustomUserAgent] = seg
		return
	}
	envs[envCustomUserAgent] = base + " " + seg
}

// 仅保留 [a-z0-9._-]，最长 32 字符。
func sanitizeAgentName(raw string) string {
	raw = strings.ToLower(raw)
	if len(raw) > maxAgentNameLen {
		raw = raw[:maxAgentNameLen]
	}
	buf := make([]byte, 0, len(raw))
	for i := 0; i < len(raw); i++ {
		c := raw[i]
		switch {
		case c >= 'a' && c <= 'z':
		case c >= '0' && c <= '9':
		case c == '.' || c == '_' || c == '-':
		default:
			continue
		}
		buf = append(buf, c)
	}
	return string(buf)
}
