package cli

import (
	"fmt"
	"strings"

	"github.com/charleszardd/yawa/internal/command"
	"github.com/charleszardd/yawa/internal/config"
	"github.com/spf13/cobra"
)

type commandOptions struct {
	typeName    string
	description string
	executable  string
	arguments   []string
	workingDir  string
	path        string
	url         string
	hotkey      string
	startDir    string
}

func (app *application) newCommandManagementCommand() *cobra.Command {
	managementCommand := &cobra.Command{Use: "command", Short: "Manage configured commands"}
	managementCommand.AddCommand(
		app.newAddCommand(),
		app.newEditCommand(),
		app.newShowCommand(),
		app.newRemoveCommand(),
	)
	return managementCommand
}

func (app *application) newAddCommand() *cobra.Command {
	options := commandOptions{}
	addCommand := &cobra.Command{
		Use:   "add <name>",
		Short: "Add a command",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			configuredCommand := options.definition(args[0])
			return app.mutateConfiguration(func(configuration *config.File) error {
				updatedCommands, err := command.Add(configuration.Commands, configuredCommand)
				configuration.Commands = updatedCommands
				return err
			})
		},
	}
	options.bind(addCommand)
	if err := addCommand.MarkFlagRequired("type"); err != nil {
		panic(err)
	}
	return addCommand
}

func (app *application) newEditCommand() *cobra.Command {
	options := commandOptions{}
	editCommand := &cobra.Command{
		Use:   "edit <name>",
		Short: "Edit a command",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.mutateConfiguration(func(configuration *config.File) error {
				registry := command.NewRegistry(configuration.Commands)
				existing, found := registry.Find(args[0])
				if !found {
					return fmt.Errorf("command %q is not configured", args[0])
				}
				options.applyChangedFlags(cmd, &existing)
				updatedCommands, err := command.Replace(configuration.Commands, args[0], existing)
				configuration.Commands = updatedCommands
				return err
			})
		},
	}
	options.bind(editCommand)
	return editCommand
}

func (app *application) newShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show <name>",
		Short: "Show a command",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			registry, err := app.loadRegistry()
			if err != nil {
				return err
			}
			configuredCommand, found := registry.Find(args[0])
			if !found {
				return fmt.Errorf("command %q is not configured", args[0])
			}
			return writeCommand(app.stdout, configuredCommand)
		},
	}
}

func (app *application) newRemoveCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a command",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return app.mutateConfiguration(func(configuration *config.File) error {
				updatedCommands, err := command.Remove(configuration.Commands, args[0])
				configuration.Commands = updatedCommands
				return err
			})
		},
	}
}

func (options *commandOptions) bind(configuredCommand *cobra.Command) {
	flags := configuredCommand.Flags()
	flags.StringVar(&options.typeName, "type", "", "action type: exec, open_path, or url")
	flags.StringVar(&options.description, "description", "", "human-readable description")
	flags.StringVar(&options.executable, "exe", "", "executable name or path")
	flags.StringArrayVar(&options.arguments, "arg", nil, "process argument; may be repeated")
	flags.StringVar(&options.workingDir, "working-dir", "", "process working directory")
	flags.StringVar(&options.path, "path", "", "file or directory to open")
	flags.StringVar(&options.url, "url", "", "HTTPS URL to open")
	flags.StringVar(&options.hotkey, "hotkey", "", "global shortcut reserved for a later phase")
	flags.StringVar(&options.startDir, "start-dir", "", "initial directory for a file or folder picker")
}

func (options commandOptions) definition(name string) command.Definition {
	return command.Definition{
		Name: name, Description: options.description, Type: command.Type(options.typeName),
		Executable: options.executable, Arguments: options.arguments, WorkingDir: options.workingDir,
		Path: options.path, URL: options.url, Hotkey: options.hotkey,
		StartDir: options.startDir,
	}
}

func (options commandOptions) applyChangedFlags(cmd *cobra.Command, configuredCommand *command.Definition) {
	if cmd.Flags().Changed("type") {
		configuredCommand.Type = command.Type(options.typeName)
	}
	if cmd.Flags().Changed("description") {
		configuredCommand.Description = options.description
	}
	if cmd.Flags().Changed("exe") {
		configuredCommand.Executable = options.executable
	}
	if cmd.Flags().Changed("arg") {
		configuredCommand.Arguments = options.arguments
	}
	if cmd.Flags().Changed("working-dir") {
		configuredCommand.WorkingDir = options.workingDir
	}
	if cmd.Flags().Changed("path") {
		configuredCommand.Path = options.path
	}
	if cmd.Flags().Changed("url") {
		configuredCommand.URL = options.url
	}
	if cmd.Flags().Changed("hotkey") {
		configuredCommand.Hotkey = options.hotkey
	}
	if cmd.Flags().Changed("start-dir") {
		configuredCommand.StartDir = options.startDir
	}
}

func (app *application) mutateConfiguration(mutation func(*config.File) error) error {
	configuration, revision, err := app.store.LoadOrDefault(app.configPath)
	if err != nil {
		return err
	}
	if err := mutation(&configuration); err != nil {
		return err
	}
	if err := app.store.Save(app.configPath, configuration, revision); err != nil {
		return err
	}
	fmt.Fprintf(app.stdout, "updated configuration: %s\n", app.configPath)
	return nil
}

func writeCommand(output interface{ Write([]byte) (int, error) }, configuredCommand command.Definition) error {
	lines := []string{
		"name: " + configuredCommand.Name,
		"description: " + configuredCommand.Description,
		"type: " + string(configuredCommand.Type),
		"executable: " + configuredCommand.Executable,
		"arguments: " + strings.Join(configuredCommand.Arguments, " | "),
		"working directory: " + configuredCommand.WorkingDir,
		"path: " + configuredCommand.Path,
		"url: " + configuredCommand.URL,
		"hotkey: " + configuredCommand.Hotkey,
		"start directory: " + configuredCommand.StartDir,
	}
	_, err := fmt.Fprintln(output, strings.Join(lines, "\n"))
	return err
}
