package telemetry

import (
	"os"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli/plugin"
)

func TestGlobalAndInnerOptOut(t *testing.T) {
	t.Setenv("ALIYUN_CLI_TELEMETRY_OPTOUT", "1")
	if !GlobalOptOut() {
		t.Fatal("expected global opt out")
	}
	t.Setenv("ALIYUN_CLI_TELEMETRY_OPTOUT", "")
	t.Setenv("ALIYUN_CLI_TELEMETRY_INNER_OPTOUT", "yes")
	if !InnerOptOut() {
		t.Fatal("expected inner opt out")
	}
}

func TestL1HardDisabled(t *testing.T) {
	if !L1HardDisabled(ProfileSnapshot{RegionID: "cn-shanghai-finance-1"}) {
		t.Fatal("finance region should disable")
	}
	if !L1HardDisabled(ProfileSnapshot{Endpoint: "https://ecs.apsara.local"}) {
		t.Fatal("apsara endpoint should disable")
	}
	if L1HardDisabled(ProfileSnapshot{RegionID: "cn-hangzhou"}) {
		t.Fatal("public region should not disable")
	}
}

func TestExtractParamNames(t *testing.T) {
	got := ExtractParamNames([]string{"rdc", "list", "--region-id", "x", "--output=json", "-q"})
	want := []string{"region-id", "output", "q"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestSanitizeSummary(t *testing.T) {
	in := "failed for /Users/alice/proj secret LTAIabcdefghijklmnop012345"
	out := SanitizeSummary(in)
	if out == in {
		t.Fatal("expected sanitization")
	}
	if contains(out, "alice") || contains(out, "LTAI") {
		t.Fatalf("leaked sensitive data: %q", out)
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func TestShouldCollectInnerBuiltInCommand(t *testing.T) {
	ok, _ := ShouldCollectInner("configure", ProfileSnapshot{}, DefaultConfig())
	if ok {
		t.Fatal("configure should not trigger inner telemetry")
	}
}

func TestDetectInnerScopeUsesDynamicReservedCommands(t *testing.T) {
	const command = "telemetry-dynamic-reserved-test"
	plugin.RegisterReservedTopLevelCommands([]string{command})
	if scope := DetectInnerScope(command); scope.Active {
		t.Fatal("dynamically registered root command should not trigger inner telemetry")
	}
}

func TestInnerAutoOnDefault(t *testing.T) {
	cfg := DefaultConfig()
	if !InnerAutoOnEnabled(cfg) {
		t.Fatal("inner auto on should default true")
	}
}

func TestAppendEvent(t *testing.T) {
	dir := t.TempDir()
	cache := GetInnerCacheDir(dir)
	ev, err := BuildEvent(BuildInput{
		Scope:      InnerScope{Active: true, Trigger: "plugin_manifest", PluginName: "p", PluginVer: "1"},
		RootArgs:   []string{"rdc", "list"},
		Profile:    ProfileSnapshot{RegionID: "cn-hangzhou"},
		InstanceID: "test-id",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := AppendEvent(cache, ev); err != nil {
		t.Fatal(err)
	}
	st := InnerCacheStats(cache)
	if st.Files != 1 || st.TotalBytes == 0 {
		t.Fatalf("unexpected stats: %+v", st)
	}
	_ = os.RemoveAll(dir)
}
