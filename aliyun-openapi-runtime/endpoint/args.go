package endpoint

import "strings"

// WithSuffixArgs rewrites explicit endpoint flags forwarded by the host CLI.
// It leaves other arguments and everything after -- untouched.
func WithSuffixArgs(args []string) []string {
	out := append([]string(nil), args...)
	for i := 0; i < len(out); i++ {
		if out[i] == "--" {
			break
		}
		name, value, inline := strings.Cut(out[i], "=")
		switch name {
		case "--body", "--body-file", "--header", "--cli-query":
			if !inline && i+1 < len(out) {
				i++
			}
		case "--endpoint", "--sts-endpoint", "-e":
			if inline {
				out[i] = name + "=" + WithSuffix(value)
			} else if i+1 < len(out) {
				i++
				out[i] = WithSuffix(out[i])
			}
		}
	}
	return out
}
