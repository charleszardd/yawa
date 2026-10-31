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
)

var commandNamePattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func Validate(configuration File) error {
	var validationErrors []error
	if configuration.General.ConfigVersion != CurrentVersion {
		validationErrors = append(validationErrors, fmt.Errorf("general.config_version must be %d", CurrentVersion))
	}

	seenNames := make(map[string]struct{}, len(configuration.Commands))
	for index, configuredCommand := range configuration.Commands {
		commandPath := fmt.Sprintf("commands[%d]", index)
		validationErrors = append(validationErrors, validateCommand(commandPath, configuredCommand, seenNames)...)
	}
	return errors.Join(validationErrors...)
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
	case command.TypeExec:
		if strings.TrimSpace(configuredCommand.Executable) == "" {
			validationErrors = append(validationErrors, fmt.Errorf("%s.executable is required for exec", commandPath))
		}
		if configuredCommand.WorkingDir != "" {
			validationErrors = append(validationErrors, validateDirectory(commandPath+".working_dir", configuredCommand.WorkingDir)...)
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
