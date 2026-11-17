package canonicalmeta

import (
	"encoding/json"
	"testing"
)

func TestImportanceAnnotationsDoNotWeakenMetadataDecode(t *testing.T) {
	var api API
	data := []byte(`{"name":"Demo","parameters":[{"raw_name":"A","type":"string","location":"query","importance_score":0},{"raw_name":"B","type":"string","location":"query","importance_score":"invalid"}]}`)
	if err := json.Unmarshal(data, &api); err != nil {
		t.Fatal(err)
	}
	views := api.LegacyTopLevelParameters()
	if got := views[0].ImportanceScore(); got == nil || *got != 0 {
		t.Fatalf("zero score lost: %v", got)
	}
	if views[1].ImportanceScore() != nil {
		t.Fatal("malformed annotation must be missing")
	}
	if err := json.Unmarshal([]byte(`{"parameters":[{"required":"bad","importance_score":"bad"}]}`), &api); err == nil {
		t.Fatal("bad business field must still fail")
	}
	var index VersionIndex
	if err := json.Unmarshal([]byte(`{"apis":{"Demo":{"importance_score":0.7},"Old":{}}}`), &index); err != nil {
		t.Fatal(err)
	}
	if index.APIs["Demo"].ImportanceScore.Value() == nil || index.APIs["Old"].ImportanceScore.Value() != nil {
		t.Fatal("version index presence lost")
	}
}

func TestLegacyImportanceUsesExactNodeWithoutParentInheritance(t *testing.T) {
	var api API
	if err := json.Unmarshal([]byte(`{"parameters":[{"importance_score":0.9}],"v1_parameters":[{"importance_score":0.2}],"v1_body_parameters":[{"importance_score":0}]}`), &api); err != nil {
		t.Fatal(err)
	}
	var field Field
	if err := json.Unmarshal([]byte(`{"importance_score":0.4}`), &field); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		view LegacyParameterView
		want float64
	}{
		{"canonical", LegacyParameterView{source: SourceCanonical, canonical: &api.Parameters[0]}, .9},
		{"field", LegacyParameterView{source: SourceField, field: &field}, .4},
		{"v1", LegacyParameterView{source: SourceV1, body: &(*api.V1Parameters)[0]}, .2},
		{"body", LegacyParameterView{source: SourceBody, body: &(*api.V1BodyParameters)[0]}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.view.ImportanceScore(); got == nil || *got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
	missing := LegacyParameterView{source: SourceField, field: &Field{}, canonical: &api.Parameters[0]}
	if missing.ImportanceScore() != nil {
		t.Fatal("nested field inherited parent score")
	}
	unknown := LegacyParameterView{source: LegacyParameterSource(-1)}
	if unknown.ImportanceScore() != nil {
		t.Fatal("unknown source must have no score")
	}
}
