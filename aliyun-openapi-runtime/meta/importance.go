package meta

import (
	"encoding/json"
	"math"
)

// ImportanceScore tolerates only malformed optional annotations, not metadata errors.
type ImportanceScore float64

func (s *ImportanceScore) UnmarshalJSON(data []byte) error {
	*s = ImportanceScore(math.NaN())
	var value *float64
	if json.Unmarshal(data, &value) == nil && value != nil && validImportance(*value) {
		*s = ImportanceScore(*value)
	}
	return nil
}

func (s ImportanceScore) MarshalJSON() ([]byte, error) {
	if !validImportance(float64(s)) {
		return []byte("null"), nil
	}
	return json.Marshal(float64(s))
}

func validImportance(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0 && value <= 1
}

func (s *ImportanceScore) Value() *float64 {
	if s == nil || !validImportance(float64(*s)) {
		return nil
	}
	value := float64(*s)
	return &value
}

func NewImportanceScore(value *float64) *ImportanceScore {
	if value == nil || !validImportance(*value) {
		return nil
	}
	score := ImportanceScore(*value)
	return &score
}
