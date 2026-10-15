package trust

// These values are intentionally empty in source. Release builds must replace
// them with generated production material. Test keys belong in _test.go files.
// An empty value makes every trust-changing operation fail closed.
var (
	BuiltinRootEnvelope     string
	BuiltinRecoveryKeysJSON string
	BuiltinRecoveryRoleJSON string
)

const (
	DiscoveryURL = "https://api.aliyun.com/.well-known/aliyun-cli-trust.json"
)

var MetadataHostAllowlist = []string{"api.aliyun.com"}
var RecoveryHostAllowlist = []string{"api.aliyun.com"}
