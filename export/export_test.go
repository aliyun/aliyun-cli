package export

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
)

func testExportProduct() *productExt {
	return &productExt{
		Code: "Demo", Version: "2026-01-01", ApiStyle: "ROA",
		Name:                 map[string]string{"en": "Demo Service", "zh": "演示服务"},
		RegionalEndpoints:    map[string]string{"cn-test": "demo.cn-test.example.com", "cn-extra": "demo.cn-extra.example.com"},
		RegionalVpcEndpoints: map[string]string{"cn-test": "demo-vpc.cn-test.example.com"},
		ApiNames:             []string{"CreateReport", "DescribeRegions"},
		LegacyExport: &legacyExportData{
			EndpointType: "regional",
			Regions: map[string]legacyRegion{
				"cn-test": {
					RegionID: "cn-test", AreaID: "china",
					RegionName: map[string]string{"en": "Test Region", "zh": "测试地域"},
					AreaName:   map[string]string{"en": "China", "zh": "中国"},
				},
			},
			APITitles: map[string]legacyAPITitle{
				"CreateReport": {En: "Create a report", Zh: "创建报表"},
			},
		},
	}
}

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

func TestExportProductWritesLegacyExecutionAndLocaleTrees(t *testing.T) {
	repo := canonicalmeta.NewRepository(os.DirFS("../canonicalmeta/testdata"))
	product := testExportProduct()
	output := t.TempDir()

	if err := exportProduct(repo, product, output); err != nil {
		t.Fatal(err)
	}
	paths := []string{
		"metadatas/demo/CreateReport.json",
		"metadatas/demo/DescribeRegions.json",
		"en-US/demo/CreateReport.json",
		"zh-CN/demo/CreateReport.json",
		"en-US/demo/version.json",
		"zh-CN/demo/version.json",
	}
	for _, relative := range paths {
		if _, err := os.Stat(filepath.Join(output, relative)); err != nil {
			t.Fatalf("missing exported file %s: %v", relative, err)
		}
	}

	var execution legacyExecAPI
	readJSONFile(t, filepath.Join(output, "metadatas/demo/DescribeRegions.json"), &execution)
	if execution.Name != "DescribeRegions" || execution.Protocol != "HTTP|HTTPS" || execution.Method != "GET|POST" {
		t.Fatalf("execution metadata = %#v", execution)
	}
	if len(execution.Parameters) != 2 || execution.Parameters[1].Name != "Tags" ||
		execution.Parameters[1].Type != "RepeatList" || len(execution.Parameters[1].SubParameters) != 2 {
		t.Fatalf("execution parameters = %#v", execution.Parameters)
	}

	var english legacyLocaleAPI
	readJSONFile(t, filepath.Join(output, "en-US/demo/CreateReport.json"), &english)
	if english.Name != "CreateReport" || len(english.Parameters) == 0 ||
		english.Parameters[0].Description != "The ID of the report to create." {
		t.Fatalf("English locale metadata = %#v", english)
	}
	var chinese legacyLocaleAPI
	readJSONFile(t, filepath.Join(output, "zh-CN/demo/CreateReport.json"), &chinese)
	if len(chinese.Parameters) == 0 || chinese.Parameters[0].Description != "要创建的报表 ID。" {
		t.Fatalf("Chinese locale metadata = %#v", chinese)
	}

	var englishVersion versionFile
	readJSONFile(t, filepath.Join(output, "en-US/demo/version.json"), &englishVersion)
	if englishVersion.Version != "2026-01-01" || englishVersion.Style != "ROA" ||
		englishVersion.APIs["CreateReport"].Title != "Create a report" ||
		englishVersion.APIs["CreateReport"].Summary != "Creates a report." {
		t.Fatalf("English version metadata = %#v", englishVersion)
	}
	var chineseVersion versionFile
	readJSONFile(t, filepath.Join(output, "zh-CN/demo/version.json"), &chineseVersion)
	if chineseVersion.APIs["CreateReport"].Title != "创建报表" ||
		chineseVersion.APIs["CreateReport"].Summary != "创建报表。" {
		t.Fatalf("Chinese version metadata = %#v", chineseVersion)
	}
}

func TestLocaleProductsAndEndpointsUseFallbacks(t *testing.T) {
	product := testExportProduct()
	endpoints := buildLocaleEndpoints(product, "zh")
	if len(endpoints) != 2 || endpoints["cn-test"].RegionName != "测试地域" ||
		endpoints["cn-test"].Public != "demo.cn-test.example.com" ||
		endpoints["cn-test"].Vpc != "demo-vpc.cn-test.example.com" ||
		endpoints["cn-extra"].RegionID != "cn-extra" {
		t.Fatalf("localized endpoints = %#v", endpoints)
	}

	fallback := &productExt{Code: "Empty", Version: "v1"}
	output := t.TempDir()
	if err := exportLocaleProductsJSON(&productSetExt{Products: []productExt{*product, *fallback}}, "en", output); err != nil {
		t.Fatal(err)
	}
	var products localeProductSet
	readJSONFile(t, filepath.Join(output, "products.json"), &products)
	if len(products.Products) != 2 || products.Products[0].Name != "Demo Service" ||
		products.Products[0].EndpointType != "regional" || products.Products[1].Name != "Empty" ||
		products.Products[1].EndpointType != "regional" {
		t.Fatalf("locale products = %#v", products.Products)
	}
}

func TestBundledProductsCanBeLoadedAndExportedWithoutLegacyFields(t *testing.T) {
	products, err := loadProductsExt()
	if err != nil {
		t.Fatal(err)
	}
	if len(products.Products) == 0 {
		t.Fatal("bundled products are empty")
	}
	output := t.TempDir()
	if err := os.MkdirAll(filepath.Join(output, "metadatas"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := exportMetadatasProductsJSON(output); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(output, "metadatas/products.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(content), "legacy_export") {
		t.Fatal("clean products export retained legacy_export")
	}
	var exported cleanProductSet
	if err := json.Unmarshal(content, &exported); err != nil {
		t.Fatal(err)
	}
	if len(exported.Products) != len(products.Products) {
		t.Fatalf("exported %d products, want %d", len(exported.Products), len(products.Products))
	}
}

func TestExportBuilderDefaultsAndWriteErrors(t *testing.T) {
	repo := canonicalmeta.NewRepository(os.DirFS("../canonicalmeta/testdata"))
	product := testExportProduct()
	product.ApiStyle = ""
	product.LegacyExport.APITitles = nil
	output := t.TempDir()
	if err := exportVersionJSON(repo, product, "en", output); err != nil {
		t.Fatal(err)
	}
	var version versionFile
	readJSONFile(t, filepath.Join(output, "demo/version.json"), &version)
	if version.Style != "rpc" || version.APIs["CreateReport"].Title != "CreateReport" {
		t.Fatalf("default version metadata = %#v", version)
	}

	api, err := repo.GetAPI("demo", "2026-01-01", "CreateReport")
	if err != nil {
		t.Fatal(err)
	}
	execution := buildMetadatasJSON(api)
	localized := buildLocaleJSON(api, "en")
	if execution.Name != "CreateReport" || localized.Name != "CreateReport" || len(execution.Parameters) == 0 {
		t.Fatalf("builder outputs = %#v / %#v", execution, localized)
	}
	if got := legacyDisplayType(api.LegacyTopLevelParameters()[0]); got != "string" {
		t.Fatalf("scalar display type = %q", got)
	}
	if err := writeJSON(filepath.Join(output, "missing", "value.json"), map[string]string{"value": "x"}); err == nil {
		t.Fatal("writeJSON unexpectedly created its missing parent")
	}
	if err := writeJSON(filepath.Join(output, "bad.json"), make(chan int)); err == nil {
		t.Fatal("writeJSON accepted an unsupported JSON value")
	}
}

func readJSONFile(t *testing.T, path string, target any) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(content, target); err != nil {
		t.Fatal(err)
	}
}
