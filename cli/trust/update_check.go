package trust

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/aliyun/aliyun-cli/v3/internal/localstate"
)

// FetchArtifactForUpdateCheck performs networking against a private cache. It
// then replays the fetched responses under the shared lock, so all trust checks
// and recovery writes use the latest foreground state without further network I/O.
func (c *Client) FetchArtifactForUpdateCheck(ctx context.Context, purpose, roleName string) (*VerifiedArtifact, error) {
	snapshot, err := c.snapshotForUpdateCheck(ctx, purpose)
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(snapshot)

	client := *c.cfg.HTTPClient
	base := client.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	fetched := &updateCheckResponses{base: base, responses: make(map[string]updateCheckResponse)}
	client.Transport = fetched
	cfg := c.cfg
	cfg.CacheDir, cfg.HTTPClient = snapshot, &client
	// Even a failed refresh can have fetched a valid recovery patch. Revalidate
	// and apply that progress to shared state rather than discarding it.
	_, _ = New(cfg).FetchArtifact(ctx, purpose, roleName)

	client.Transport = updateCheckReplayTransport{fetched}
	cfg.CacheDir = c.cfg.CacheDir
	return New(cfg).FetchArtifact(ctx, purpose, roleName)
}

func (c *Client) snapshotForUpdateCheck(ctx context.Context, purpose string) (string, error) {
	anchor, err := c.bootstrapRoot(c.cfg.Now().UTC())
	if err != nil {
		return "", err
	}
	discovery, err := c.discoveryURL()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(c.cfg.CacheDir, 0700); err != nil {
		return "", err
	}
	unlock, err := localstate.Lock(ctx, filepath.Join(c.cfg.CacheDir, ".metadata.lock"))
	if err != nil {
		return "", err
	}
	defer unlock()
	dir, err := os.MkdirTemp("", "aliyun-trust-check-*")
	if err != nil {
		return "", err
	}
	// Only this repository and its Discovery record are needed for validation.
	// Sibling repositories are inspected/cleared by recovery during commit;
	// copying their potentially large indexes would prolong the shared lock.
	roots := []string{
		filepath.Join(c.cfg.CacheDir, "discovery", digest(discovery)+".json"),
		filepath.Join(c.cfg.CacheDir, digest(anchor.Authority), digest(c.cfg.SourceBase)),
	}
	copyEntry := func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if path == dir && entry.IsDir() {
			return filepath.SkipDir
		}
		if entry.Name() == ".metadata.lock" || (strings.HasPrefix(entry.Name(), ".") && !entry.IsDir()) {
			return nil
		}
		if strings.HasSuffix(entry.Name(), ".envelope.json") && entry.Name() != purpose+".envelope.json" {
			return nil
		}
		relative, err := filepath.Rel(c.cfg.CacheDir, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dir, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0700)
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("unsupported trust cache file type")
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		data, err := io.ReadAll(io.LimitReader(f, maxResponseBytes+1))
		_ = f.Close()
		if err != nil {
			return err
		}
		if len(data) > maxResponseBytes {
			return fmt.Errorf("trust cache file exceeds size limit")
		}
		if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0600)
	}
	for _, root := range roots {
		err = filepath.WalkDir(root, copyEntry)
		if err != nil && !os.IsNotExist(err) {
			_ = os.RemoveAll(dir)
			return "", err
		}
	}
	return dir, nil
}

type updateCheckResponse struct {
	response *http.Response
	body     []byte
	err      error
}

func (r updateCheckResponse) replay(req *http.Request) (*http.Response, error) {
	if r.err != nil {
		return nil, r.err
	}
	response := *r.response
	response.Header = r.response.Header.Clone()
	response.Trailer = r.response.Trailer.Clone()
	response.Body = io.NopCloser(bytes.NewReader(r.body))
	response.Request = req
	return &response, nil
}

type updateCheckResponses struct {
	base      http.RoundTripper
	mu        sync.Mutex
	responses map[string]updateCheckResponse
}

func (f *updateCheckResponses) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := f.base.RoundTrip(req)
	r := updateCheckResponse{err: err}
	if response == nil && err == nil {
		r.err = fmt.Errorf("empty trust response")
	}
	if response != nil {
		copy := *response
		copy.Header, copy.Trailer = response.Header.Clone(), response.Trailer.Clone()
		copy.Request, copy.Body = nil, nil
		r.response = &copy
		if response.Body != nil {
			r.body, err = io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
			_ = response.Body.Close()
			if r.err == nil {
				r.err = err
			}
		}
	}
	f.mu.Lock()
	f.responses[req.Method+" "+req.URL.String()] = r
	f.mu.Unlock()
	return r.replay(req)
}

type updateCheckReplayTransport struct{ fetched *updateCheckResponses }

func (t updateCheckReplayTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.fetched.mu.Lock()
	r, ok := t.fetched.responses[req.Method+" "+req.URL.String()]
	t.fetched.mu.Unlock()
	if !ok {
		// A concurrent foreground update may need a root/URL not seen during
		// prefetch. Fail closed or use verified LKG; never go online under lock.
		return nil, fmt.Errorf("trust metadata changed during update check")
	}
	return r.replay(req)
}
