package util

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func collectAgentEnvs() []string {
	envs := make([]string, 0, len(knownAgentEnv)+1)
	envs = append(envs, agentEnvProposal)
	for _, item := range knownAgentEnv {
		envs = append(envs, item.env)
	}
	return envs
}

var allAgentEnvs = collectAgentEnvs()

func snapshotAndUnsetAgentEnvs(t *testing.T) {
	t.Helper()
	saved := make(map[string]string, len(allAgentEnvs))
	for _, k := range allAgentEnvs {
		if v, ok := os.LookupEnv(k); ok {
			saved[k] = v
		}
		_ = os.Unsetenv(k)
	}
	t.Cleanup(func() {
		for _, k := range allAgentEnvs {
			_ = os.Unsetenv(k)
		}
		for k, v := range saved {
			_ = os.Setenv(k, v)
		}
	})
}

func TestDetectAgentName_NoEnv(t *testing.T) {
	snapshotAndUnsetAgentEnvs(t)
	assert.Equal(t, "", DetectAgentName())
	assert.Equal(t, "", GetAgentUserAgentSegment())
}

func TestDetectAgentName_KnownEnvs(t *testing.T) {
	for _, c := range knownAgentEnv {
		t.Run(c.env, func(t *testing.T) {
			snapshotAndUnsetAgentEnvs(t)
			_ = os.Setenv(c.env, "1")
			assert.Equal(t, c.name, DetectAgentName())
			assert.Equal(t, "agent/"+c.name, GetAgentUserAgentSegment())
		})
	}
}

func TestDetectAgentName_EmptyValueIgnored(t *testing.T) {
	snapshotAndUnsetAgentEnvs(t)
	_ = os.Setenv("CURSOR_AGENT", "")
	assert.Equal(t, "", DetectAgentName())
}

func TestDetectAgentName_ProposalAgentEnvWins(t *testing.T) {
	snapshotAndUnsetAgentEnvs(t)
	_ = os.Setenv(agentEnvProposal, "Goose")
	_ = os.Setenv("CURSOR_AGENT", "1")
	assert.Equal(t, "goose", DetectAgentName())
}

func TestDetectAgentName_ProposalSanitize(t *testing.T) {
	snapshotAndUnsetAgentEnvs(t)
	_ = os.Setenv(agentEnvProposal, "  My-Agent.v1_x  ")
	assert.Equal(t, "my-agent.v1_x", DetectAgentName())
}

func TestDetectAgentName_ProposalRejectsAllInvalidChars(t *testing.T) {
	snapshotAndUnsetAgentEnvs(t)
	_ = os.Setenv(agentEnvProposal, "$$$ ###")
	_ = os.Setenv("CURSOR_AGENT", "1")
	assert.Equal(t, "cursor", DetectAgentName(),
		"完全无效的 AGENT 值应回退到专有变量")
}

func TestDetectAgentName_ProposalTruncated(t *testing.T) {
	snapshotAndUnsetAgentEnvs(t)
	long := "abcdefghij1234567890ABCDEFGHIJ-_.zzzzz"
	_ = os.Setenv(agentEnvProposal, long)
	got := DetectAgentName()
	assert.LessOrEqual(t, len(got), maxAgentNameLen)
}

func TestDetectAgentName_PriorityOrder(t *testing.T) {
	snapshotAndUnsetAgentEnvs(t)
	_ = os.Setenv("CLAUDECODE", "1")
	_ = os.Setenv("CURSOR_AGENT", "1")
	assert.Equal(t, "cursor", DetectAgentName(),
		"CURSOR_AGENT 在表中靠前，应优先返回")
}

func TestMergeAgentSegmentIntoPluginEnvs_NoAgent(t *testing.T) {
	snapshotAndUnsetAgentEnvs(t)
	envs := map[string]string{}
	MergeAgentSegmentIntoPluginEnvs(envs)
	_, ok := envs[envCustomUserAgent]
	assert.False(t, ok, "未检测到 agent 时不应写入任何 env")
}

func TestMergeAgentSegmentIntoPluginEnvs_NilMap(t *testing.T) {
	snapshotAndUnsetAgentEnvs(t)
	_ = os.Setenv("CURSOR_AGENT", "1")
	assert.NotPanics(t, func() {
		MergeAgentSegmentIntoPluginEnvs(nil)
	})
}

func TestMergeAgentSegmentIntoPluginEnvs_FreshEnv(t *testing.T) {
	snapshotAndUnsetAgentEnvs(t)
	_ = os.Unsetenv(envCustomUserAgent)
	_ = os.Setenv("CURSOR_AGENT", "1")
	envs := map[string]string{}
	MergeAgentSegmentIntoPluginEnvs(envs)
	assert.Equal(t, "agent/cursor", envs[envCustomUserAgent])
}

func TestMergeAgentSegmentIntoPluginEnvs_PreservesParentEnv(t *testing.T) {
	snapshotAndUnsetAgentEnvs(t)
	_ = os.Setenv("CURSOR_AGENT", "1")
	t.Setenv(envCustomUserAgent, "skill/foo")
	envs := map[string]string{}
	MergeAgentSegmentIntoPluginEnvs(envs)
	assert.Equal(t, "skill/foo agent/cursor", envs[envCustomUserAgent],
		"已 export 的 ALIBABA_CLOUD_USER_AGENT 必须保留并在末尾追加 agent 段")
}

func TestMergeAgentSegmentIntoPluginEnvs_RuntimeEnvWinsOverParent(t *testing.T) {
	snapshotAndUnsetAgentEnvs(t)
	_ = os.Setenv("CURSOR_AGENT", "1")
	t.Setenv(envCustomUserAgent, "from-parent")
	envs := map[string]string{
		envCustomUserAgent: "from-runtime",
	}
	MergeAgentSegmentIntoPluginEnvs(envs)
	assert.Equal(t, "from-runtime agent/cursor", envs[envCustomUserAgent],
		"envs 中已有值时优先使用 envs 中的值")
}
