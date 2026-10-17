package meta

import (
	"testing"
	"time"
)

func TestMetadataReadTimeout(t *testing.T) {
	for _, tc := range []struct {
		ms   int64
		want time.Duration
	}{
		{55000, 57 * time.Second}, {60000, 62 * time.Second}, {10001, 12001 * time.Millisecond},
		{0, 0}, {-1, 0}, {1<<63 - 1, 0},
	} {
		if got := MetadataReadTimeout(tc.ms); got != tc.want {
			t.Errorf("%d: got %v, want %v", tc.ms, got, tc.want)
		}
	}
}
