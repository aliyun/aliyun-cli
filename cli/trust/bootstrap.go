package trust

// Public trust anchors shipped with every build, including Homebrew and other
// source builds that do not read GitHub Actions variables. These values are
// public; Root and Recovery private keys are not here. Linker -X on the
// Base64 variables overrides this material for a later key rotation.
var (
	BuiltinRootEnvelope = `{"payload":"eyJhdXRob3JpdHkiOiJodHRwczovL2FwaS5hbGl5dW4uY29tIiwiZXhwaXJlc19hdCI6IjIwMjctMTAtMjJUMDg6MjY6NDRaIiwia2V5cyI6W3siYWxnb3JpdGhtIjoiRWQyNTUxOSIsImtleV9pZCI6InJvb3QtMjAyNjEwMjIwODI2NDQtYSIsInB1YmxpY19rZXkiOiJwZzJSVEFpUjZlVkUwT1dESUxENGJjOUhoWDl2eFlhZnl2blNTTW9ZeVdZPSJ9LHsiYWxnb3JpdGhtIjoiRWQyNTUxOSIsImtleV9pZCI6InJvb3QtMjAyNjEwMjIwODI2NDQtYiIsInB1YmxpY19rZXkiOiJBc3J0QVFVQjVGZHlqdTZ1Ym4rWDJKYjlFN0JZODhJQitHQm1Bam13UzNJPSJ9LHsiYWxnb3JpdGhtIjoiRWQyNTUxOSIsImtleV9pZCI6InJvb3QtMjAyNjEwMjIwODI2NDQtYyIsInB1YmxpY19rZXkiOiJFZUozNGllMWtsb01lSThWVnk5eDNqd0VSNGpvSGJseU02TEFoMEZCdlA4PSJ9XSwicm9sZXMiOnsiYXJ0aWZhY3Qta2V5cyI6eyJrZXlfaWRzIjpbInJvb3QtMjAyNjEwMjIwODI2NDQtYSIsInJvb3QtMjAyNjEwMjIwODI2NDQtYiIsInJvb3QtMjAyNjEwMjIwODI2NDQtYyJdLCJ0aHJlc2hvbGQiOjJ9LCJyb290Ijp7ImtleV9pZHMiOlsicm9vdC0yMDI2MTAyMjA4MjY0NC1hIiwicm9vdC0yMDI2MTAyMjA4MjY0NC1iIiwicm9vdC0yMDI2MTAyMjA4MjY0NC1jIl0sInRocmVzaG9sZCI6Mn19LCJzY2hlbWFfdmVyc2lvbiI6MSwidHlwZSI6InJvb3QiLCJ2ZXJzaW9uIjoxfQ==","schema_version":1,"signatures":[{"algorithm":"Ed25519","key_id":"root-20261022082644-a","signature":"B28e5+TN+x7DTsDOEfyFVAp+2XWS/ALz1fuXvLnCaGiP3UITZZ7UNEvc+OND+vp5mydvzIz58cfZYAV6VJZ/Dw=="},{"algorithm":"Ed25519","key_id":"root-20261022082644-b","signature":"Be/Tak78DjnJ0nxRiVY6WrmQLgvXJArj+3qpgom1A6F76wGGIJvXcjfOncvDFwll/YgIQUFRPE5vPd7emO5ADQ=="},{"algorithm":"Ed25519","key_id":"root-20261022082644-c","signature":"AhU/k2EH4rs6r8tidEDMRyKdiTVO3XvsOj5JSv56cjDj/XwnLYOB32AkRQ7i3mn8O+Hxr5ou0yjXG+opYhlWCA=="}]}
`
	BuiltinRecoveryKeysJSON = `[{"algorithm":"Ed25519","key_id":"recovery-20261022082644-a","public_key":"gFLVDeHAyIDXJUTh7h62PQNhj5VnBio9SlT7wy92nRU="},{"algorithm":"Ed25519","key_id":"recovery-20261022082644-b","public_key":"H1XbB3a34lmDqFjY+ZadTkl3x+XQJ2h/wQUN5pV3UNQ="},{"algorithm":"Ed25519","key_id":"recovery-20261022082644-c","public_key":"EH3Q4EQvBeVXNgyStSQzpBAl45IuzOpjVkBLC+JfGbQ="}]
`
	BuiltinRecoveryRoleJSON = `{"key_ids":["recovery-20261022082644-a","recovery-20261022082644-b","recovery-20261022082644-c"],"threshold":2}
`
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
