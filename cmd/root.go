package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

func newRootCmd() *cobra.Command {
	var rootCmd = &cobra.Command{
		Use:           "minio-config-cli",
		Short:         "The minio-config-cli is a CLI tool for declaratively managing minio configurations",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	return rootCmd
}

func Execute() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	rootCmd := newRootCmd()
	rootCmd.AddCommand(newImportCmd())
	rootCmd.AddCommand(newVersionCmd())
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
