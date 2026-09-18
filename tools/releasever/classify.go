package releasever

import (
	"fmt"
	"strings"

	"golang.org/x/mod/semver"
)

// Kind is the release class used by the publish pipeline.
type Kind string

const (
	KindStable     Kind = "stable"
	KindPrerelease Kind = "prerelease"
)

// Plan is the publish policy for a version. Prerelease versions must not
// update stable/latest pointers or be marked as the official GitHub release.
type Plan struct {
	Kind               Kind
	UpdateLatest       bool
	WriteStableVersion bool
	MarkOfficial       bool
}

// Classify reports whether raw is a stable or prerelease version.
// raw may be a tag (v3.5.1) or the version string used by finish_release (3.5.1).
// Any legal semver prerelease (beta, beta.1, rc.1, alpha, …) is prerelease.
// Versions that are not legal semver are rejected.
func Classify(raw string) (Kind, error) {
	canonical, err := canonicalVersion(raw)
	if err != nil {
		return "", err
	}
	if semver.Prerelease(canonical) != "" {
		return KindPrerelease, nil
	}
	return KindStable, nil
}

// ReleasePlan is the single publish decision for a version.
func ReleasePlan(raw string) (Plan, error) {
	kind, err := Classify(raw)
	if err != nil {
		return Plan{}, err
	}
	if kind == KindPrerelease {
		return Plan{Kind: KindPrerelease}, nil
	}
	return Plan{
		Kind:               KindStable,
		UpdateLatest:       true,
		WriteStableVersion: true,
		MarkOfficial:       true,
	}, nil
}

func canonicalVersion(raw string) (string, error) {
	v := strings.TrimSpace(raw)
	if v == "" {
		return "", fmt.Errorf("invalid release version %q: empty", raw)
	}
	if strings.ContainsAny(v, " \t\r\n") {
		return "", fmt.Errorf("invalid release version %q", raw)
	}
	if strings.HasPrefix(v, "v") {
		v = v[1:]
	}
	if v == "" {
		return "", fmt.Errorf("invalid release version %q", raw)
	}
	candidate := "v" + v
	if !semver.IsValid(candidate) || !hasMajorMinorPatch(candidate) {
		return "", fmt.Errorf("invalid release version %q", raw)
	}
	return candidate, nil
}

// hasMajorMinorPatch requires vMAJOR.MINOR.PATCH. golang.org/x/mod/semver
// accepts shorter forms such as v3.5; those must not be published.
func hasMajorMinorPatch(version string) bool {
	core := strings.TrimPrefix(version, "v")
	if i := strings.IndexAny(core, "-+"); i >= 0 {
		core = core[:i]
	}
	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
	}
	return true
}
