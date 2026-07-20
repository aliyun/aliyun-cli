package export

import (
	"os"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
)

func TestExportProductFailsWhenCanonicalAPIMissing(t *testing.T) {
	repo := canonicalmeta.NewRepository(os.DirFS("../canonicalmeta/testdata"))
	product := &productExt{
		Code:     "demo",
		Version:  "2026-01-01",
		ApiStyle: "rpc",
		ApiNames: []string{"MissingAPI"},
	}

	err := exportProduct(repo, product, t.TempDir())
	if err == nil {
		t.Fatal("expected missing canonical API to fail export")
	}
	if !strings.Contains(err.Error(), "MissingAPI") {
		t.Fatalf("expected error to mention MissingAPI, got %s", err)
	}
}
