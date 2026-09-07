// Package trust implements artifact authenticity checks for aliyun-cli:
// Ed25519 detached signatures over plugin indexes / upgrade manifests,
// freshness (monotonic version + expiry), and optional root.json key delegation.
//
// See docs/zh-CN/artifact-trust-signing.md for the design.
package trust
