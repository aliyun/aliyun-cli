package trust

// These values are intentionally empty in source. Release builds must replace
// them with generated production material. Test keys belong in _test.go files.
// An empty value makes every trust-changing operation fail closed.
var (
	BuiltinRootEnvelope       string
	BuiltinRecoveryKeysJSON   string
	BuiltinRecoveryRoleJSON   string
	BuiltinRootEnvelopeBase64 string
	BuiltinRecoveryKeysBase64 string
	BuiltinRecoveryRoleBase64 string
)

const (
	DiscoveryURLCN   = "https://api.aliyun.com/.well-known/aliyun-cli-trust.json"
	DiscoveryURLIntl = "https://api.alibabacloud.com/.well-known/aliyun-cli-trust.json"
	// DiscoveryURL remains as the compatibility name for the default cn site.
	DiscoveryURL = DiscoveryURLCN
)

var MetadataHostAllowlist = []string{
	"api.aliyun.com", "api.alibabacloud.com", "aliyuncli.alicdn.com",
}
var RecoveryHostAllowlist = []string{
	"api.aliyun.com", "api.alibabacloud.com", "aliyuncli.alicdn.com",
}
