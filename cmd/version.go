package cmd

import (
	"encoding/json"
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"
)

type versionInfo struct {
	Version   string
	GoVersion string
}

func newVersionCmd() *cobra.Command {
	var versionCmd = &cobra.Command{
		Use:     "version",
		Short:   "Print the version of minio-config-cli",
		Example: "minio-config-cli version",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			info, ok := debug.ReadBuildInfo()
			if !ok {
				panic("failed to read build info")
			}
			asJSON, err := json.Marshal(
				versionInfo{
					Version:   info.Main.Version,
					GoVersion: info.GoVersion,
				})
			if err != nil {
				return fmt.Errorf("failed to marshal version info: %w", err)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", asJSON)
			if err != nil {
				return fmt.Errorf("failed to print version info: %w", err)
			}
			return nil
		},
	}
	return versionCmd
}
