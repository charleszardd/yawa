package command

import "testing"

func TestRegistryFindIsCaseInsensitive(t *testing.T) {
	registry := NewRegistry([]Definition{{Name: "open-code", Type: TypeExec}})

	configuredCommand, found := registry.Find("OPEN-CODE")
	if !found || configuredCommand.Name != "open-code" {
		t.Fatalf("Find() = (%v, %v)", configuredCommand, found)
	}
}

func TestRegistryAllReturnsCopy(t *testing.T) {
	registry := NewRegistry([]Definition{{Name: "first"}})
	commands := registry.All()
	commands[0].Name = "modified"

	configuredCommand, found := registry.Find("first")
	if !found || configuredCommand.Name != "first" {
		t.Fatal("All() exposed registry storage")
	}
}
