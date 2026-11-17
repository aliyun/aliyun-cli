package format

import (
	"testing"
)

func TestBackendTimeoutSurvivesJSONDecoding(t *testing.T) {
	for _, tc := range []struct {
		name, field string
		want        int64
	}{
		{"old metadata without timeout", "", 0},
		{"milliseconds preserved", `,"timeout_ms":12001`, 12001},
		{"final timeout", `,"timeout_ms":57000`, 57000},
		{"final cap", `,"timeout_ms":62000`, 62000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api, err := DecodeAPIJSON([]byte(`{"name":"CreateDBInstance","operation":{"action":"CreateDBInstance","api_version":"2014-08-15","method":"POST"}`+tc.field+`}`), "test")
			if err != nil {
				t.Fatal(err)
			}
			if api.TimeoutMS != tc.want {
				t.Fatalf("metadata timeout = %d, want %d", api.TimeoutMS, tc.want)
			}
		})
	}
}
