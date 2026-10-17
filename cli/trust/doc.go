// Package trust implements artifact authenticity for aliyun-cli.
//
// Design A: builtin Root/Recovery, versioned N.root.json, timestamp, recovery patches.
// Design B+: fixed Discovery for location + builtin Root for authenticity of
// artifact-keys and the Root chain Discovery points to.
package trust
