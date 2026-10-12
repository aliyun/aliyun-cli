package format

import (
	"testing"
	"testing/fstest"

	"github.com/aliyun/aliyun-openapi-runtime/source/storage"
)

func TestJSONImportancePreservesZeroAndDropsOnlyInvalidAnnotations(t *testing.T) {
	api, err := DecodeAPIJSON([]byte(`{"name":"Demo","operation":{},"parameters":[{"name":"one","raw_name":"One","type":"string","importance_score":0},{"name":"two","type":"string","importance_score":"bad"},{"name":"nested","type":"object","fields":[{"name":"child","type":"string","importance_score":0.8}]}]}`), "test")
	if err != nil {
		t.Fatal(err)
	}
	if score := api.Parameters[0].ImportanceScore; score == nil || *score != 0 {
		t.Fatal("zero missing")
	}
	if api.Parameters[1].ImportanceScore != nil {
		t.Fatal("invalid score not missing")
	}
	if score := api.Parameters[2].Fields[0].ImportanceScore; score == nil || *score != .8 {
		t.Fatal("nested score lost")
	}
	if api.Parameters[2].ImportanceScore != nil {
		t.Fatal("parent score invented")
	}
	if _, err := DecodeAPIJSON([]byte(`{"operation":{},"parameters":[{"type":"array","importance_score":"bad"}]}`), "broken"); err == nil {
		t.Fatal("broken shape was accepted")
	}
	vol, err := storage.NewFSStorage(fstest.MapFS{"demo/v1/version.json": &fstest.MapFile{Data: []byte(`{"apis":{"Demo":{"cmd_name":"demo","importance_score":0},"Old":{},"Bad":{"importance_score":false}}}`)}}, "").Open("demo")
	if err != nil {
		t.Fatal(err)
	}
	defer vol.Close()
	index, err := NewJSONFormat().DecodeIndex(vol, "v1")
	if err != nil {
		t.Fatal(err)
	}
	if index.Entries["Demo"].ImportanceScore == nil || *index.Entries["Demo"].ImportanceScore != 0 {
		t.Fatal("index zero missing")
	}
	if index.Entries["Old"].ImportanceScore != nil || index.Entries["Bad"].ImportanceScore != nil {
		t.Fatal("index missing score invented")
	}
}
