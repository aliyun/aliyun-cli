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

func TestImportanceScoreRoundTripAndMissing(t *testing.T) {
	if NewImportanceScore(nil) != nil {
		t.Fatal("missing score became present")
	}
	for _, value := range []float64{0, .5, 1} {
		score := NewImportanceScore(&value)
		data, err := json.Marshal(score)
		if err != nil || score.Value() == nil || *score.Value() != value {
			t.Fatalf("valid score did not survive: %v %s %v", value, data, err)
		}
		var decoded ImportanceScore
		if err := json.Unmarshal(data, &decoded); err != nil || decoded.Value() == nil || *decoded.Value() != value {
			t.Fatalf("round trip: %s %v", data, err)
		}
	}
	for _, value := range []float64{math.NaN(), math.Inf(1), -1, 2} {
		data, err := json.Marshal(ImportanceScore(value))
		if err != nil || string(data) != "null" {
			t.Fatalf("invalid annotation must marshal as null: %s %v", data, err)
		}
	}
}
