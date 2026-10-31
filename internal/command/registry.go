package command

import "strings"

type Registry struct {
	commands    []Definition
	commandsMap map[string]Definition
}

func NewRegistry(commands []Definition) Registry {
	commandsCopy := append([]Definition(nil), commands...)
	commandsMap := make(map[string]Definition, len(commandsCopy))
	for _, configuredCommand := range commandsCopy {
		commandsMap[normalizeName(configuredCommand.Name)] = configuredCommand
	}

	return Registry{commands: commandsCopy, commandsMap: commandsMap}
}

func (registry Registry) All() []Definition {
	return append([]Definition(nil), registry.commands...)
}

func (registry Registry) Find(name string) (Definition, bool) {
	configuredCommand, found := registry.commandsMap[normalizeName(name)]
	return configuredCommand, found
}

func normalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
