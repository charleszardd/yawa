package command

import "testing"

func TestCommandMutationsDoNotModifyInput(t *testing.T) {
	original := []Definition{{Name: "one", Type: TypeExec, Executable: "one.exe"}}

	added, err := Add(original, Definition{Name: "two", Type: TypeURL, URL: "https://example.com"})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	replaced, err := Replace(added, "one", Definition{Name: "one", Type: TypeExec, Executable: "updated.exe"})
	if err != nil {
		t.Fatalf("Replace() error = %v", err)
	}
	removed, err := Remove(replaced, "two")
	if err != nil {
		t.Fatalf("Remove() error = %v", err)
	}

	if original[0].Executable != "one.exe" {
		t.Fatal("mutation changed its input")
	}
	if len(removed) != 1 || removed[0].Executable != "updated.exe" {
		t.Fatalf("result = %+v", removed)
	}
}

func TestAddRejectsDuplicateNameIgnoringCase(t *testing.T) {
	_, err := Add([]Definition{{Name: "editor"}}, Definition{Name: "EDITOR"})
	if err == nil {
		t.Fatal("Add() error = nil")
	}
}
