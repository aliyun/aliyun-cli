package openapi

import (
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

type endpointFailWriter struct{ remaining int }

func (w *endpointFailWriter) Write(p []byte) (int, error) {
	if w.remaining == 0 {
		return 0, io.ErrClosedPipe
	}
	w.remaining--
	return len(p), nil
}
func TestEndpointHelpWriteErrors(t *testing.T) {
	for _, n := range []int{0, 1, 2} {
		doc := &machineHelpProductDocument{unsupportedRegion: "unknown", Endpoints: []machineHelpEndpoint{{Endpoint: "global"}}}
		require.ErrorIs(t, renderProductEndpoints(&endpointFailWriter{n}, doc), io.ErrClosedPipe)
	}
	require.Equal(t, 1, endpointCellWidth("e\u0301"))
}
