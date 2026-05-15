package mock

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingFileReturnsEmptyRecords(t *testing.T) {
	records, err := Load(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("Load returned %d records, want 0", len(records))
	}
}

func TestLoadEmptyFileReturnsEmptyRecords(t *testing.T) {
	for name, content := range map[string]string{
		"empty":      "",
		"whitespace": " \n\t ",
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "mocks.json")
			if err := os.WriteFile(path, []byte(content), 0600); err != nil {
				t.Fatalf("WriteFile: %v", err)
			}

			records, err := Load(path)
			if err != nil {
				t.Fatalf("Load returned error: %v", err)
			}
			if len(records) != 0 {
				t.Fatalf("Load returned %d records, want 0", len(records))
			}
		})
	}
}

func TestLoadInvalidNullReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mocks.json")
	if err := os.WriteFile(path, []byte("null"), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if records, err := Load(path); err == nil {
		t.Fatalf("Load(null) = %#v, nil error", records)
	}
}

func TestLoadInvalidNullRecordReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mocks.json")
	if err := os.WriteFile(path, []byte("[null]"), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if records, err := Load(path); err == nil {
		t.Fatalf("Load([null]) = %#v, nil error", records)
	}
}

func TestLoadInvalidEmptyCommandReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mocks.json")
	if err := os.WriteFile(path, []byte(`[{"name":"bad","cmd":"","exit_code":0,"stdout":"","stderr":"","times":0}]`), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if records, err := Load(path); err == nil {
		t.Fatalf("Load(empty cmd) = %#v, nil error", records)
	}
}

func TestSaveCreatesParentDirsAndWritesIndentedJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "mock", "mocks.json")
	if err := Save(path, []Record{{
		Name:     "saved",
		Cmd:      "ecs *",
		ExitCode: 2,
		Stdout:   "ok",
		Stderr:   "",
		Times:    10,
	}}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	want := `[
  {
    "name": "saved",
    "cmd": "ecs *",
    "exit_code": 2,
    "stdout": "ok",
    "stderr": "",
    "times": 10
  }
]
`
	if string(data) != want {
		t.Fatalf("saved content = %q, want %q", string(data), want)
	}
}

func TestReadInputAcceptsFlatSingleObjectAndArray(t *testing.T) {
	one, err := DecodeInput([]byte(`{"name":"one","cmd":"ecs *","exit_code":0,"stdout":"","stderr":"","times":0}`))
	if err != nil || len(one) != 1 || one[0].Name != "one" {
		t.Fatalf("DecodeInput(single) = %#v, %v", one, err)
	}

	many, err := DecodeInput([]byte(`[{"name":"a","cmd":"ecs a","exit_code":0,"stdout":"a","stderr":"","times":1},{"name":"b","cmd":"ecs b","exit_code":2,"stdout":"","stderr":"b","times":0}]`))
	if err != nil || len(many) != 2 || many[1].Name != "b" {
		t.Fatalf("DecodeInput(array) = %#v, %v", many, err)
	}
}

func TestDecodeInputRejectsMissingRequiredFlatFields(t *testing.T) {
	input := `{"name":"missing","cmd":"ecs *","exit_code":0,"stdout":"","stderr":""}`

	if records, err := DecodeInput([]byte(input)); err == nil {
		t.Fatalf("DecodeInput(missing times) = %#v, nil error", records)
	}
}

func TestDecodeInputRejectsNestedOutputFormat(t *testing.T) {
	input := `{"name":"nested","cmd":"ecs *","output":{"exit_code":0,"stdout":"","stderr":""},"times":1}`

	if records, err := DecodeInput([]byte(input)); err == nil {
		t.Fatalf("DecodeInput(nested output) = %#v, nil error", records)
	}
}

func TestDecodeInputEmptyInputReturnsError(t *testing.T) {
	if records, err := DecodeInput([]byte(" \n\t ")); err == nil {
		t.Fatalf("DecodeInput(empty) = %#v, nil error", records)
	}
}

func TestDecodeInputRejectsNullRecords(t *testing.T) {
	for name, input := range map[string]string{
		"single": "null",
		"array":  "[null]",
	} {
		t.Run(name, func(t *testing.T) {
			if records, err := DecodeInput([]byte(input)); err == nil {
				t.Fatalf("DecodeInput(%s) = %#v, nil error", input, records)
			}
		})
	}
}

func TestAppendAndClearRecords(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mocks.json")
	if err := Append(path, []Record{{Name: "first", Cmd: "ecs *", Times: 0}, {Name: "second", Cmd: "ram *", Times: 0}}); err != nil {
		t.Fatalf("Append first: %v", err)
	}
	if err := Append(path, []Record{{Name: "third", Cmd: "vpc *", Times: 0}}); err != nil {
		t.Fatalf("Append second: %v", err)
	}

	records, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(records) != 3 || records[0].Name != "first" || records[2].Name != "third" {
		t.Fatalf("records order = %#v", records)
	}

	if err := Clear(path); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "[]\n" {
		t.Fatalf("clear content = %q, want [] newline", string(data))
	}
}

func TestValidateRecordRejectsMissingNameAndNegativeTimes(t *testing.T) {
	if err := ValidateRecord(Record{Cmd: "ecs *", Times: 0}); err == nil {
		t.Fatal("ValidateRecord missing name returned nil error")
	}
	if err := ValidateRecord(Record{Name: "bad", Cmd: "ecs *", Times: -1}); err == nil {
		t.Fatal("ValidateRecord negative times returned nil error")
	}
	if err := ValidateRecord(Record{Name: "ok", Cmd: "ecs *", Times: 0}); err != nil {
		t.Fatalf("ValidateRecord valid record returned error: %v", err)
	}
}

func TestRemoveByNameRemovesFirstMatchAndErrorsWhenMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mocks.json")
	if err := Save(path, []Record{
		{Name: "first", Cmd: "ecs first", Times: 0},
		{Name: "second", Cmd: "ecs second", Times: 0},
		{Name: "first", Cmd: "ecs first again", Times: 0},
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := RemoveByName(path, "first"); err != nil {
		t.Fatalf("RemoveByName first: %v", err)
	}
	records, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(records) != 2 || records[0].Name != "second" || records[1].Cmd != "ecs first again" {
		t.Fatalf("records after RemoveByName = %#v", records)
	}

	if err := RemoveByName(path, "missing"); err == nil {
		t.Fatal("RemoveByName missing returned nil error")
	}
}

func TestRemoveByIndexRemovesRecordAndRejectsOutOfRange(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mocks.json")
	if err := Save(path, []Record{
		{Name: "zero", Cmd: "ecs zero", Times: 0},
		{Name: "one", Cmd: "ecs one", Times: 0},
		{Name: "two", Cmd: "ecs two", Times: 0},
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := RemoveByIndex(path, 1); err != nil {
		t.Fatalf("RemoveByIndex 1: %v", err)
	}
	records, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(records) != 2 || records[0].Name != "zero" || records[1].Name != "two" {
		t.Fatalf("records after RemoveByIndex = %#v", records)
	}

	for _, index := range []int{-1, 2} {
		if err := RemoveByIndex(path, index); err == nil {
			t.Fatalf("RemoveByIndex(%d) returned nil error", index)
		}
	}
}

func TestLoadLenientReturnsEmptyForMalformedData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mocks.json")
	if err := os.WriteFile(path, []byte("{bad"), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	records := LoadLenient(path)

	if len(records) != 0 {
		t.Fatalf("LoadLenient returned %d records, want 0", len(records))
	}
}

func TestAppendLenientRecoversMalformedData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mocks.json")
	if err := os.WriteFile(path, []byte("{bad"), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := AppendLenient(path, []Record{{Name: "new", Cmd: "ecs new", Times: 0}}); err != nil {
		t.Fatalf("AppendLenient: %v", err)
	}

	records, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(records) != 1 || records[0].Name != "new" {
		t.Fatalf("records = %#v, want recovered record", records)
	}
}

func TestSaveSetsExistingFileModeToPrivate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mocks.json")
	if err := os.WriteFile(path, []byte("[]\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatalf("Chmod setup: %v", err)
	}

	if err := Save(path, []Record{{Name: "private", Cmd: "ecs *", Times: 0}}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("file mode = %v, want %v", got, os.FileMode(0600))
	}
}
