package cli

import (
	"context"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/charleszardd/yawa/internal/command"
	"github.com/charleszardd/yawa/internal/config"
	"github.com/charleszardd/yawa/internal/launcher"
	"github.com/spf13/cobra"
)

type application struct {
	configPath string
	loader     config.Loader
	dispatcher command.Dispatcher
	stdout     io.Writer
}

func Execute(version string, stdout io.Writer, stderr io.Writer) error {
	configPath, err := config.DefaultPath()
	if err != nil {
		return err
	}

	app := &application{
		configPath: configPath,
		loader:     config.Loader{},
		dispatcher: command.NewDispatcher(launcher.Windows{}),
		stdout:     stdout,
	}
	rootCommand := app.newRootCommand(version)
	rootCommand.SetOut(stdout)
	rootCommand.SetErr(stderr)
	rootCommand.SilenceErrors = true
	rootCommand.SilenceUsage = true
	return rootCommand.Execute()
}

func (app *application) newRootCommand(version string) *cobra.Command {
	rootCommand := &cobra.Command{
		Use:           "yawa",
		Short:         "Launch configured Windows applications, paths, and URLs",
		Version:       version,
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	rootCommand.PersistentFlags().StringVar(&app.configPath, "config", app.configPath, "configuration file path")
	rootCommand.AddCommand(
		app.newListCommand(),
		app.newRunCommand(),
		app.newValidateCommand(),
		app.newConfigCommand(),
	)
	return rootCommand
}

func (app *application) newListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List configured commands",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			registry, err := app.loadRegistry()
			if err != nil {
				return err
			}
			writer := tabwriter.NewWriter(app.stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(writer, "NAME\tHOTKEY\tTYPE\tDESCRIPTION")
			for _, configuredCommand := range registry.All() {
				fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n", configuredCommand.Name, configuredCommand.Hotkey, configuredCommand.Type, configuredCommand.Description)
			}
			return writer.Flush()
		},
	}
}

func (app *application) newRunCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "run <command-name>",
		Short: "Run a configured command",
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
			return app.dispatcher.Dispatch(context.Background(), configuredCommand)
		},
	}
}

func (app *application) newValidateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate the configuration file",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			if _, err := app.loadRegistry(); err != nil {
				return err
			}
			fmt.Fprintf(app.stdout, "configuration is valid: %s\n", app.configPath)
			return nil
		},
	}
}

func (app *application) newConfigCommand() *cobra.Command {
	configCommand := &cobra.Command{Use: "config", Short: "Inspect configuration settings"}
	configCommand.AddCommand(&cobra.Command{
		Use:   "path",
		Short: "Print the active configuration file path",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			_, err := fmt.Fprintln(app.stdout, app.configPath)
			return err
		},
	})
	return configCommand
}

func (app *application) loadRegistry() (command.Registry, error) {
	configuration, err := app.loader.Load(app.configPath)
	if err != nil {
		return command.Registry{}, err
	}
	if err := config.Validate(configuration); err != nil {
		return command.Registry{}, fmt.Errorf("validate configuration: %w", err)
	}
	return command.NewRegistry(configuration.Commands), nil
}
