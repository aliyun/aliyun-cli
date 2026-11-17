package endpoint

import (
	"context"
	"io"
	"net/http"
)

// NewRequest applies the endpoint suffix before an unsigned HTTP request is
// built. Signed SDK requests must resolve their endpoint before signing.
func NewRequest(method, address string, body io.Reader) (*http.Request, error) {
	return NewRequestWithContext(context.Background(), method, address, body)
}

// NewRequestWithContext preserves the caller's context and body.
func NewRequestWithContext(ctx context.Context, method, address string, body io.Reader) (*http.Request, error) {
	if err := Validate(); err != nil {
		return nil, err
	}
	return http.NewRequestWithContext(ctx, method, WithSuffix(address), body)
}

// Get uses the caller's HTTP client, including its timeout, transport and
// redirect policy. A nil client selects http.DefaultClient.
func Get(client *http.Client, address string) (*http.Response, error) {
	req, err := NewRequest(http.MethodGet, address, nil)
	if err != nil {
		return nil, err
	}
	if client == nil {
		client = http.DefaultClient
	}
	return client.Do(req)
}

// HTTPGet is the suffix-aware equivalent of http.Get.
func HTTPGet(address string) (*http.Response, error) {
	return Get(http.DefaultClient, address)
}
