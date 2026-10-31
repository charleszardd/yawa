package command

import (
	"fmt"
	"strings"
)

func Add(commands []Definition, newCommand Definition) ([]Definition, error) {
	if _, found := NewRegistry(commands).Find(newCommand.Name); found {
		return nil, fmt.Errorf("command %q already exists", newCommand.Name)
	}
	return append(append([]Definition(nil), commands...), newCommand), nil
}

func Replace(commands []Definition, name string, replacement Definition) ([]Definition, error) {
	updated := append([]Definition(nil), commands...)
	for index := range updated {
		if strings.EqualFold(updated[index].Name, name) {
			updated[index] = replacement
			return updated, nil
		}
	}
	return nil, fmt.Errorf("command %q is not configured", name)
}

func Remove(commands []Definition, name string) ([]Definition, error) {
	updated := make([]Definition, 0, len(commands))
	found := false
	for _, configuredCommand := range commands {
		if strings.EqualFold(configuredCommand.Name, name) {
			found = true
			continue
		}
		updated = append(updated, configuredCommand)
	}
	if !found {
		return nil, fmt.Errorf("command %q is not configured", name)
	}
	return updated, nil
}
