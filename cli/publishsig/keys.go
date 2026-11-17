package publishsig

import "strings"

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
// SignatureVersion is the publisher key generation this binary follows.
// Upgrade installs the newest release whose public_keys list contains it.
const SignatureVersion = "1"

// SupportedPublicKeyVersions are the publisher key generations embedded in
// this binary. Each stable release copies this list into /versions. A package
// that trusts more than one generation lists each generation itself; support
// is not inferred from where the version sits between two other releases.
//
// On a bridge release, set this to the old and new generations and set
// SignatureVersion to the new generation. Older clients still select the
// bridge because it lists their generation. The next upgrade follows the new one.
var SupportedPublicKeyVersions = []string{SignatureVersion}

// VersionsFormat is the JSON document published at /versions.
const VersionsFormat = "aliyun-cli-versions v1"

// DescribePublicKeys is the note stored beside a package's public key generations.
// It does not name the signature algorithm.
func DescribePublicKeys(versions []string) string {
	switch len(versions) {
	case 0:
		return "publisher key generations"
	case 1:
		return "publisher key generation " + versions[0]
	default:
		return "publisher key generations " + strings.Join(versions, ", ")
	}
}

var embeddedPublicKeys = []string{
	"pzznzV/mIilgTwd5sNg1uzTnQLKOCeQM2MhztBrtx+A=",
}
