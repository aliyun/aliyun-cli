package legacyapi

import (
	"io"
	"net/http"
)

// Response retains the HTTP bytes and multi-value headers before SDK parsing.
type Response struct {
	Body       []byte
	StatusCode int
	Headers    http.Header
}

func NewResponse() *Response                     { return &Response{} }
func (r *Response) GetHttpContentString() string { return string(r.Body) }
func (r *Response) GetHttpContentBytes() []byte  { return r.Body }

// ReadResponse buffers a response without interpreting the payload as JSON.
func ReadResponse(r *Response, h *http.Response) error {
	defer h.Body.Close()
	body, err := io.ReadAll(h.Body)
	r.Body, r.StatusCode, r.Headers = body, h.StatusCode, h.Header.Clone()
	return err
}
