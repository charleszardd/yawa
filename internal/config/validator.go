package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/charleszardd/yawa/internal/command"
	"github.com/charleszardd/yawa/internal/hotkey"
)

var commandNamePattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func Validate(configuration File) error {
	var validationErrors []error
	if configuration.General.ConfigVersion != CurrentVersion {
		validationErrors = append(validationErrors, fmt.Errorf("general.config_version must be %d", CurrentVersion))
	}

	seenNames := make(map[string]struct{}, len(configuration.Commands))
	seenHotkeys := make(map[string]string, len(configuration.Commands))
	for index, configuredCommand := range configuration.Commands {
		commandPath := fmt.Sprintf("commands[%d]", index)
		validationErrors = append(validationErrors, validateCommand(commandPath, configuredCommand, seenNames)...)
		validationErrors = append(validationErrors, validateHotkey(commandPath, configuredCommand, seenHotkeys)...)
	}
	return errors.Join(validationErrors...)
}

func validateHotkey(commandPath string, configuredCommand command.Definition, seen map[string]string) []error {
	if configuredCommand.Hotkey == "" {
		return nil
	}
	binding, err := hotkey.Parse(configuredCommand.Hotkey)
	if err != nil {
		return []error{fmt.Errorf("%s.hotkey: %w", commandPath, err)}
	}
	canonical := hotkey.Canonical(binding)
	if existing, duplicate := seen[canonical]; duplicate {
		return []error{fmt.Errorf("%s.hotkey duplicates command %q", commandPath, existing)}
	}
	seen[canonical] = configuredCommand.Name
	return nil
}

func validateCommand(commandPath string, configuredCommand command.Definition, seenNames map[string]struct{}) []error {
	var validationErrors []error
	if !commandNamePattern.MatchString(configuredCommand.Name) {
		validationErrors = append(validationErrors, fmt.Errorf("%s.name must match %s", commandPath, commandNamePattern))
	}
	normalizedName := strings.ToLower(configuredCommand.Name)
	if _, exists := seenNames[normalizedName]; exists {
		validationErrors = append(validationErrors, fmt.Errorf("%s.name %q is duplicated", commandPath, configuredCommand.Name))
	}
	seenNames[normalizedName] = struct{}{}

	switch configuredCommand.Type {
	case command.TypeExec, command.TypePickFolder, command.TypePickFile:
		if strings.TrimSpace(configuredCommand.Executable) == "" {
			validationErrors = append(validationErrors, fmt.Errorf("%s.executable is required for %s", commandPath, configuredCommand.Type))
		}
		if configuredCommand.WorkingDir != "" {
			validationErrors = append(validationErrors, validateDirectory(commandPath+".working_dir", configuredCommand.WorkingDir)...)
		}
		if configuredCommand.Type != command.TypeExec {
			if !containsPlaceholder(configuredCommand.Arguments) {
				validationErrors = append(validationErrors, fmt.Errorf("%s.args must contain %s", commandPath, command.SelectedPathPlaceholder))
			}
			if configuredCommand.StartDir != "" {
				validationErrors = append(validationErrors, validateDirectory(commandPath+".start_dir", configuredCommand.StartDir)...)
			}
		}
	case command.TypeOpenPath:
		if strings.TrimSpace(configuredCommand.Path) == "" {
			validationErrors = append(validationErrors, fmt.Errorf("%s.path is required for open_path", commandPath))
		}
	case command.TypeURL:
		validationErrors = append(validationErrors, validateURL(commandPath+".url", configuredCommand.URL)...)
	default:
		validationErrors = append(validationErrors, fmt.Errorf("%s.type %q is unsupported", commandPath, configuredCommand.Type))
	}
	return validationErrors
}

func containsPlaceholder(arguments []string) bool {
	for _, argument := range arguments {
		if strings.Contains(argument, command.SelectedPathPlaceholder) {
			return true
		}
	}
	return false
}

func validateDirectory(fieldPath string, directoryPath string) []error {
	expandedPath := os.ExpandEnv(directoryPath)
	info, err := os.Stat(filepath.Clean(expandedPath))
	if err != nil {
		return []error{fmt.Errorf("%s is not accessible: %w", fieldPath, err)}
	}
	if !info.IsDir() {
		return []error{fmt.Errorf("%s must reference a directory", fieldPath)}
	}
	return nil
}

func validateURL(fieldPath string, rawURL string) []error {
	parsedURL, err := url.ParseRequestURI(rawURL)
	if err != nil || parsedURL.Scheme != "https" || parsedURL.Host == "" {
		return []error{fmt.Errorf("%s must be an absolute https URL", fieldPath)}
	}
	return nil
}
