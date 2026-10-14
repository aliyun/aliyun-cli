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
var embeddedPublicKeys = []string{
	"pzznzV/mIilgTwd5sNg1uzTnQLKOCeQM2MhztBrtx+A=",
}
