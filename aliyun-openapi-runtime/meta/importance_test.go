package meta

import (
	"encoding/json"
	"math"
	"testing"
)

func TestImportanceScoreOptionalDecode(t *testing.T) {
	for _, input := range []string{"0", "0.9", "1", "null", " null ", `"0.9"`, "true", "{}", "-1", "2"} {
		t.Run(input, func(t *testing.T) {
			var score *ImportanceScore
			if err := json.Unmarshal([]byte(input), &score); err != nil {
				t.Fatal(err)
			}
			valid := input == "0" || input == "0.9" || input == "1"
			if (score.Value() != nil) != valid {
				t.Fatalf("score %s: got %v", input, score.Value())
			}
		})
	}
	for _, value := range []float64{math.NaN(), math.Inf(1), -0.1, 1.1} {
		if NewImportanceScore(&value) != nil {
			t.Fatalf("accepted %v", value)
		}
	}
}
