package cmd

import "github.com/spf13/cobra"

func New() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "conxec",
		Short: "conxec is a CLI tool for debuging running container.",
	}

	rootCmd.AddCommand(ExecCmd())

	return rootCmd
}
