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
