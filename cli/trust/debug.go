package trust

import (
	"fmt"
	"os"
	"strings"
)

func trustDebugEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("ALIBABA_CLOUD_CLI_TRUST_DEBUG"))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func trustDebugf(format string, args ...any) {
	if !trustDebugEnabled() {
		return
	}
	fmt.Fprintf(os.Stderr, "trust: "+format+"\n", args...)
}

func rootSummary(root Root) string {
	ids := make([]string, 0, len(root.Keys))
	for _, key := range root.Keys {
		ids = append(ids, key.KeyID)
	}
	role := root.Roles["root"]
	return fmt.Sprintf(
		"version=%d key_ids=%s threshold=%d/%d",
		root.Version, strings.Join(ids, ","), role.Threshold, len(role.KeyIDs),
	)
}
