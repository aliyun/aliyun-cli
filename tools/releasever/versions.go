package releasever

import "strings"

// AppendStableVersion adds one stable version to the history file.
// Existing lines are kept in order, including lines this version does not
// understand, so a newer history format is not stripped by this release.
// The version is recorded once, without a leading v, and the result ends
// with a newline.
func AppendStableVersion(existing []byte, raw string) ([]byte, error) {
	kind, err := Classify(raw)
	if err != nil {
		return nil, err
	}
	if kind != KindStable {
		return nil, errPrerelease(raw)
	}
	canonical, err := canonicalVersion(raw)
	if err != nil {
		return nil, err
	}
	line := strings.TrimPrefix(canonical, "v")

	text := string(existing)
	if text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	for _, existingLine := range strings.Split(strings.TrimSuffix(text, "\n"), "\n") {
		if existingLine == "" && text == "" {
			continue
		}
		if sameStableVersion(existingLine, line) {
			return []byte(text), nil
		}
	}
	if text == "\n" {
		text = ""
	}
	return []byte(text + line + "\n"), nil
}

func sameStableVersion(existingLine, want string) bool {
	canonical, err := canonicalVersion(existingLine)
	if err != nil {
		return false
	}
	if semverPrerelease(canonical) {
		return false
	}
	return strings.TrimPrefix(canonical, "v") == want
}

func semverPrerelease(canonical string) bool {
	return strings.Contains(strings.TrimPrefix(canonical, "v"), "-")
}

func errPrerelease(raw string) error {
	return &prereleaseError{raw: raw}
}

type prereleaseError struct {
	raw string
}

func (e *prereleaseError) Error() string {
	return "refuse to record prerelease " + e.raw
}
