package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func New(version, commit string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:     "conxec",
		Version: fmt.Sprintf("%s (commit: %s)", version, commit),
		Short:   "conxec is a CLI tool for debuging running container.",
	}

	rootCmd.AddCommand(ExecCmd())

	return rootCmd
}
