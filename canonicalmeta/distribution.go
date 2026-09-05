package canonicalmeta

import "strings"

const (
	DistributionGo      = "go"
	DistributionMeta    = "meta"
	DistributionOpenAPI = "openapi"
)

const distributionSep = "|"

func ParseDistribution(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, distributionSep)
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		tok := strings.ToLower(strings.TrimSpace(part))
		if tok == "" {
			continue
		}
		if _, ok := seen[tok]; ok {
			continue
		}
		seen[tok] = struct{}{}
		out = append(out, tok)
	}
	return out
}

func HasDistribution(raw, kind string) bool {
	kind = strings.ToLower(strings.TrimSpace(kind))
	if kind == "" {
		return false
	}
	for _, tok := range ParseDistribution(raw) {
		if tok == kind {
			return true
		}
	}
	return false
}

func (p ProductEntry) HasDistribution(kind string) bool {
	return HasDistribution(p.Distribution, kind)
}
