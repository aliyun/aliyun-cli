package argparser

import (
	"errors"
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/meta"
)

func TestUnknownFlagImportanceKeepsAliasesAndFlagOwnership(t *testing.T) {
	zero, high := 0.0, .9
	params := []meta.Parameter{
		{Name: "first", RawName: "First", Type: meta.TypeString, Options: []string{"--first", "--alias-first"}, ImportanceScore: &zero},
		{Name: "second", RawName: "Second", Type: meta.TypeString, Options: []string{"--second"}, ImportanceScore: &high},
		{Name: "body", RawName: "Body", Type: meta.TypeString, Options: []string{"--body"}, ImportanceScore: &high},
	}
	_, err := Parse(params, []string{"--missing"})
	var unknown *UnknownFlagError
	if !errors.As(err, &unknown) {
		t.Fatalf("got %T %v", err, err)
	}
	if unknown.ImportanceScores["first"] != 0 || unknown.ImportanceScores["alias-first"] != 0 || len(unknown.ImportanceScores) != 3 {
		t.Fatal(unknown.ImportanceScores)
	}
	if !unknown.GlobalNames["body"] {
		t.Fatal("reserved flag ownership lost")
	}
	if _, exists := unknown.ImportanceScores["body"]; exists {
		t.Fatal("API score borrowed by reserved flag")
	}
	_, err = Parse(params, []string{"--First", "x"})
	if !errors.As(err, &unknown) {
		t.Fatal("RawName became an accepted alias")
	}
}
