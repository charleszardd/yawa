package cli

import (
	"errors"

	"github.com/spf13/cobra"
)

const (
	ExitGeneral   = 1
	ExitUsage     = 2
	ExitNotFound  = 3
	ExitExecution = 4
	ExitAgent     = 5
)

type exitError struct {
	code  int
	cause error
}

func (err exitError) Error() string { return err.cause.Error() }
func (err exitError) Unwrap() error { return err.cause }

func withExitCode(code int, err error) error {
	if err == nil {
		return nil
	}
	return exitError{code: code, cause: err}
}

func ExitCode(err error) int {
	var codedError exitError
	if errors.As(err, &codedError) {
		return codedError.code
	}
	return ExitGeneral
}

func noArgs(cmd *cobra.Command, args []string) error {
	return withExitCode(ExitUsage, cobra.NoArgs(cmd, args))
}

func exactArgs(count int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		return withExitCode(ExitUsage, cobra.ExactArgs(count)(cmd, args))
	}
}
