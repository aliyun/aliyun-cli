package publishsig

// Trusted publisher keys for aliyun upgrade.
//
// A signature is accepted only when it verifies under one of these keys and
// the signed asset name matches the file the client asked to install.
// To rotate: add the new public key, ship a CLI release, then switch the
// release secret. Remove the old key only after that release is the oldest
// client you still want to accept upgrades from.
//
// The matching secret is not in this repository. Release jobs read it from
// the GitHub Actions secret CLI_RELEASE_SIGNING_KEY.
//
// SignatureVersion is the publisher generation this binary will upgrade to.
// Each stable release records the generation of the key that signed it in
// https://aliyuncli.alicdn.com/versions. Upgrade installs the newest release
// with this same value.
//
// A bridge release is signed by the previous key, so set
// CLI_RELEASE_SIGNATURE_VERSION to that previous generation while this
// constant in the bridge binary is already the next one. Older clients then
// stop at the bridge, and the next aliyun upgrade follows the new generation.
const SignatureVersion = "1"

// SignatureVersionDescription is the note stored beside SignatureVersion.
const SignatureVersionDescription = "Ed25519 publisher key generation 1"

// VersionsFormat is the JSON document published at /versions.
const VersionsFormat = "aliyun-cli-versions v1"

// DescribeSignatureVersion is the note recorded for one publisher generation.
func DescribeSignatureVersion(version string) string {
	if version == SignatureVersion {
		return SignatureVersionDescription
	}
	return "publisher signature version " + version
}

var embeddedPublicKeys = []string{
	"pzznzV/mIilgTwd5sNg1uzTnQLKOCeQM2MhztBrtx+A=",
}
