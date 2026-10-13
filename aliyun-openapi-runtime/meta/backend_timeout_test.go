package meta

import (
	"testing"
	"time"
)

func TestBackendReadTimeout(t *testing.T) {
	for _, tc := range []struct {
		ms   int64
		want time.Duration
	}{
		{55000, 65 * time.Second}, {80000, 90 * time.Second}, {1, 10001 * time.Millisecond},
		{0, 0}, {-1, 0}, {1<<63 - 1, 0},
	} {
		if got := BackendReadTimeout(tc.ms); got != tc.want {
			t.Errorf("%d: got %v, want %v", tc.ms, got, tc.want)
		}
	}
}
