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
		RunE: func(_ *cobra.Command, _ []string) error {
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
			fmt.Println(string(asJSON)) //nolint:forbidigo // we just want to print the version, no logging
			return nil
		},
	}
	return versionCmd
}
