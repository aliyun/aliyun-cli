package plugin

import (
	"errors"
	"fmt"
	"net/url"
)

// safePluginPackageURL returns a diagnostic URL without credentials or
// caller-controlled query and fragment data. The original URL remains
// unchanged and must be used for the actual download.
func safePluginPackageURL(value *url.URL) string {
	if value == nil {
		return "<unknown>"
	}

	safe := *value
	safe.User = nil
	safe.RawQuery = ""
	safe.ForceQuery = false
	safe.Fragment = ""
	safe.RawFragment = ""
	return safe.String()
}

// safePluginDownloadError avoids rendering url.Error.URL, which contains the
// original package URL and may therefore include credentials or signed query
// parameters. Wrapping the underlying cause preserves errors.Is/errors.As
// behavior for transport failures without retaining the credential-bearing
// URL in the rendered error.
func safePluginDownloadError(err error, displayURL string) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) && urlErr.Err != nil {
		return fmt.Errorf("download plugin package from %s: %w", displayURL, urlErr.Err)
	}
	return fmt.Errorf("download plugin package from %s: %w", displayURL, err)
}
