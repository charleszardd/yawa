package ui

import (
	"context"
	"fmt"
	"sort"

	"github.com/charleszardd/yawa/internal/appcatalog"
	"github.com/charleszardd/yawa/internal/command"
	"github.com/charleszardd/yawa/internal/config"
	"github.com/charleszardd/yawa/internal/dialogs"
)

type Service struct {
	configPath string
	store      config.Store
	dispatcher command.Dispatcher
}

type Snapshot struct {
	ConfigPath string               `json:"configPath"`
	Commands   []command.Definition `json:"commands"`
}

func NewService(configPath string, dispatcher command.Dispatcher) *Service {
	return &Service{configPath: configPath, store: config.NewStore(), dispatcher: dispatcher}
}

func (service *Service) Applications() []appcatalog.Application {
	applications := appcatalog.InstalledApplications()
	if applications == nil {
		return []appcatalog.Application{}
	}
	return applications
}

func (service *Service) ResolveExecutable(value string) appcatalog.ExecutableStatus {
	return appcatalog.ResolveExecutable(value)
}

func (service *Service) BrowseExecutable() (string, error) {
	path, selected, err := (dialogs.Picker{}).SelectExecutable(context.Background())
	if err != nil {
		return "", err
	}
	if !selected {
		return "", nil
	}
	return path, nil
}

func (service *Service) Load() (Snapshot, error) {
	configuration, revision, err := service.store.LoadOrDefault(service.configPath)
	if err != nil {
		return Snapshot{}, err
	}
	if revision == (config.Revision{}) {
		if err := service.store.Save(service.configPath, configuration, revision); err != nil {
			return Snapshot{}, err
		}
	}
	commands := make([]command.Definition, len(configuration.Commands))
	copy(commands, configuration.Commands)
	sort.Slice(commands, func(left, right int) bool { return commands[left].Name < commands[right].Name })
	return Snapshot{ConfigPath: service.configPath, Commands: commands}, nil
}

func (service *Service) Save(configuredCommand command.Definition, originalName string) error {
	configuration, revision, err := service.store.LoadOrDefault(service.configPath)
	if err != nil {
		return err
	}
	if originalName == "" {
		configuration.Commands, err = command.Add(configuration.Commands, configuredCommand)
	} else {
		configuration.Commands, err = command.Replace(configuration.Commands, originalName, configuredCommand)
	}
	if err != nil {
		return err
	}
	return service.store.Save(service.configPath, configuration, revision)
}

func (service *Service) Delete(name string) error {
	configuration, revision, err := service.store.Load(service.configPath)
	if err != nil {
		return err
	}
	configuration.Commands, err = command.Remove(configuration.Commands, name)
	if err != nil {
		return err
	}
	return service.store.Save(service.configPath, configuration, revision)
}

func (service *Service) Run(name string) error {
	configuration, _, err := service.store.Load(service.configPath)
	if err != nil {
		return err
	}
	if err := config.Validate(configuration); err != nil {
		return fmt.Errorf("validate configuration: %w", err)
	}
	configuredCommand, found := command.NewRegistry(configuration.Commands).Find(name)
	if !found {
		return fmt.Errorf("command %q is not configured", name)
	}
	return service.dispatcher.Dispatch(context.Background(), configuredCommand)
}
