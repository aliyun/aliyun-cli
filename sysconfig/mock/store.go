package mock

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func Load(path string) ([]Record, error) {
	return load(path, true)
}

func LoadLenient(path string) []Record {
	records, err := load(path, false)
	if err != nil {
		return []Record{}
	}
	return records
}

func load(path string, strict bool) ([]Record, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Record{}, nil
		}
		return nil, err
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return []Record{}, nil
	}

	var records []Record
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, err
	}
	if records == nil {
		if strict {
			return nil, fmt.Errorf("mock data must be an array of records")
		}
		return []Record{}, nil
	}
	if err := validateRecords(records); strict && err != nil {
		return nil, err
	} else if err != nil {
		return []Record{}, nil
	}
	return records, nil
}

func Save(path string, records []Record) error {
	if records == nil {
		records = []Record{}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0600); err != nil {
		return err
	}
	return os.Chmod(path, 0600)
}

func DecodeInput(data []byte) ([]Record, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("mock input is empty")
	}

	if trimmed[0] == '[' {
		var records []Record
		if err := json.Unmarshal(trimmed, &records); err != nil {
			return nil, err
		}
		if err := validateRecords(records); err != nil {
			return nil, err
		}
		return records, nil
	}

	var record Record
	if err := json.Unmarshal(trimmed, &record); err != nil {
		return nil, err
	}
	if err := validateRecord(record); err != nil {
		return nil, err
	}
	return []Record{record}, nil
}

func validateRecords(records []Record) error {
	for _, record := range records {
		if err := validateRecord(record); err != nil {
			return err
		}
	}
	return nil
}

func ValidateRecord(record Record) error {
	return validateRecord(record)
}

func validateRecord(record Record) error {
	if record.Name == "" {
		return fmt.Errorf("mock record name is required")
	}
	if record.Cmd == "" {
		return fmt.Errorf("mock record cmd is required")
	}
	if record.Times < 0 {
		return fmt.Errorf("mock record times must be greater than or equal to 0")
	}
	return nil
}

func Append(path string, records []Record) error {
	current, err := Load(path)
	if err != nil {
		return err
	}
	return Save(path, append(current, records...))
}

func AppendLenient(path string, records []Record) error {
	current := LoadLenient(path)
	return Save(path, append(current, records...))
}

func Clear(path string) error {
	return Save(path, []Record{})
}

func RemoveByName(path string, name string) error {
	records := LoadLenient(path)
	for i, record := range records {
		if record.Name == name {
			return Save(path, append(records[:i], records[i+1:]...))
		}
	}
	return fmt.Errorf("mock record not found: name %s", name)
}

func RemoveByIndex(path string, index int) error {
	records := LoadLenient(path)
	if index < 0 || index >= len(records) {
		return fmt.Errorf("mock record index out of range: %d", index)
	}
	return Save(path, append(records[:index], records[index+1:]...))
}
