package cli

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"text/tabwriter"

	"github.com/charleszardd/yawa/internal/command"
	"github.com/charleszardd/yawa/internal/config"
	"github.com/charleszardd/yawa/internal/dialogs"
	"github.com/charleszardd/yawa/internal/hotkey"
	"github.com/charleszardd/yawa/internal/launcher"
	"github.com/charleszardd/yawa/internal/logging"
	"github.com/spf13/cobra"
)

type application struct {
	configPath string
	store      config.Store
	launcher   command.Launcher
	dispatcher command.Dispatcher
	stdout     io.Writer
	logger     *slog.Logger
}

func Execute(version string, stdout io.Writer, stderr io.Writer) error {
	configPath, err := config.DefaultPath()
	if err != nil {
		return err
	}

	windowsLauncher := launcher.Windows{}
	logger, logFile, err := logging.Open()
	if err != nil {
		return err
	}
	defer logFile.Close()
	app := &application{
		configPath: configPath,
		store:      config.NewStore(),
		launcher:   windowsLauncher,
		dispatcher: command.NewDispatcher(windowsLauncher, dialogs.Picker{}),
		stdout:     stdout,
		logger:     logger,
	}
	rootCommand := app.newRootCommand(version)
	rootCommand.SetOut(stdout)
	rootCommand.SetErr(stderr)
	rootCommand.SilenceErrors = true
	rootCommand.SilenceUsage = true
	rootCommand.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return withExitCode(ExitUsage, err)
	})
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
		app.newCommandManagementCommand(),
		app.newAgentCommand(),
	)
	return rootCommand
}

func (app *application) newAgentCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "agent",
		Short: "Run the global hotkey agent",
		Args:  noArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			registry, err := app.loadRegistry()
			if err != nil {
				return withExitCode(ExitUsage, err)
			}
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			fmt.Fprintln(app.stdout, "yawa agent is running; press Ctrl+C to stop")
			return withExitCode(ExitAgent, hotkey.NewAgent(app.dispatcher, app.stdout, app.logger).Run(ctx, registry))
		},
	}
}

func (app *application) newListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List configured commands",
		Args:  noArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			registry, err := app.loadRegistry()
			if err != nil {
				return withExitCode(ExitUsage, err)
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
		Args:  exactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			registry, err := app.loadRegistry()
			if err != nil {
				return withExitCode(ExitUsage, err)
			}
			configuredCommand, found := registry.Find(args[0])
			if !found {
				return withExitCode(ExitNotFound, fmt.Errorf("command %q is not configured", args[0]))
			}
			app.logger.Info("command started", "command", configuredCommand.Name, "type", configuredCommand.Type)
			if err := app.dispatcher.Dispatch(context.Background(), configuredCommand); err != nil {
				app.logger.Error("command failed", "command", configuredCommand.Name, "type", configuredCommand.Type, "error", err)
				return withExitCode(ExitExecution, err)
			}
			app.logger.Info("command launched", "command", configuredCommand.Name, "type", configuredCommand.Type)
			return nil
		},
	}
}

func (app *application) newValidateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate the configuration file",
		Args:  noArgs,
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
		Args:  noArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			_, err := fmt.Fprintln(app.stdout, app.configPath)
			return err
		},
	})
	configCommand.AddCommand(&cobra.Command{
		Use:   "edit",
		Short: "Open the configuration file in its associated editor",
		Args:  noArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := app.ensureConfigExists(); err != nil {
				return err
			}
			return app.launcher.OpenPath(app.configPath)
		},
	})
	return configCommand
}

func (app *application) loadRegistry() (command.Registry, error) {
	configuration, _, err := app.store.Load(app.configPath)
	if err != nil {
		return command.Registry{}, err
	}
	if err := config.Validate(configuration); err != nil {
		return command.Registry{}, fmt.Errorf("validate configuration: %w", err)
	}
	return command.NewRegistry(configuration.Commands), nil
}

func (app *application) ensureConfigExists() error {
	configuration, revision, err := app.store.LoadOrDefault(app.configPath)
	if err != nil {
		return err
	}
	if revision != (config.Revision{}) {
		return nil
	}
	return app.store.Save(app.configPath, configuration, revision)
}
