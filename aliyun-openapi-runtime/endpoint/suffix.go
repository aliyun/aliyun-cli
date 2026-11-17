// Package endpoint applies the process-wide Alibaba Cloud endpoint suffix.
package endpoint

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

const EnvSuffix = "ALIBABA_CLOUD_ENDPOINT_SUFFIX"

// Validate rejects unsupported suffixes before credentials or requests are sent.
func Validate() error {
	switch suffix := strings.TrimSpace(os.Getenv(EnvSuffix)); suffix {
	case "", "aliyuncs.com", "aliyuncs.alibaba":
		return nil
	default:
		return fmt.Errorf("invalid %s %q: expected aliyuncs.com or aliyuncs.alibaba", EnvSuffix, suffix)
	}
}

// WithSuffix rewrites only a terminal .aliyuncs.com hostname suffix, preserving
// schemes, ports, paths and queries. Call Validate before sending requests;
// unsupported values never change the destination. No shared state is mutated.
func WithSuffix(address string) string {
	if strings.TrimSpace(os.Getenv(EnvSuffix)) != "aliyuncs.alibaba" || address == "" {
		return address
	}
	prefix := ""
	if !strings.Contains(address, "://") {
		prefix = "//"
	}
	u, err := url.Parse(prefix + address)
	if err != nil {
		return address
	}
	host := u.Hostname()
	const old = ".aliyuncs.com"
	if !strings.HasSuffix(strings.ToLower(host), old) {
		return address
	}
	host = host[:len(host)-len(old)] + ".aliyuncs.alibaba"
	if port := u.Port(); port != "" {
		host += ":" + port
	}
	u.Host = host
	return strings.TrimPrefix(u.String(), prefix)
}
