package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newRootCmd() *cobra.Command {
	var rootCmd = &cobra.Command{
		Use:   "minio-config-cli",
		Short: "The minio-config-cli is a CLI tool for declaratively managing minio configurations",
	}
	return rootCmd
}

func Execute() {
	rootCmd := newRootCmd()
	rootCmd.AddCommand(newImportCmd())
	rootCmd.AddCommand(newVersionCmd())
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
