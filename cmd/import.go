package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"

	"github.com/minio/madmin-go/v3"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/spf13/cobra"
	"github.com/yardenshoham/minio-config-cli/pkg/reconcile"
)

func newImportCmd() *cobra.Command {
	var importFileLocations []string
	var importCmd = &cobra.Command{
		Use:     "import MINIO_URL ACCESS_KEY SECRET_KEY",
		Short:   "Import configuration from the specified files",
		Example: "minio-config-cli import http://localhost:9000 minioadmin minioadmin --import-file-location=config.yaml",
		Args:    cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			url, err := url.Parse(args[0])
			if err != nil {
				return fmt.Errorf("failed to parse minio url: %v", err)
			}
			secure := url.Scheme == "https"
			creds := credentials.NewStaticV4(args[1], args[2], "")
			madminClient, err := madmin.NewWithOptions(url.Host, &madmin.Options{
				Secure: secure,
				Creds:  creds,
			})
			minioClient, err := minio.New(url.Host, &minio.Options{
				Creds:  creds,
				Secure: secure,
			})
			if err != nil {
				return fmt.Errorf("failed to create madmin client: %v", err)
			}
			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
			ctx := context.Background()
			for _, importFileLocation := range importFileLocations {
				file, err := os.Open(importFileLocation)
				if err != nil {
					return fmt.Errorf("failed to open file %s: %v", importFileLocation, err)
				}
				defer file.Close()
				config, err := reconcile.LoadConfig(file)
				if err != nil {
					return fmt.Errorf("failed to load config from file %s: %v", importFileLocation, err)
				}
				err = reconcile.Import(logger.With("file", importFileLocation), ctx, madminClient, minioClient, *config)
				if err != nil {
					return fmt.Errorf("failed to import from file %s: %v", importFileLocation, err)
				}
			}
			return nil
		},
	}
	const importFileLocationsFlagName = "import-file-location"
	importCmd.Flags().StringSliceVarP(&importFileLocations, importFileLocationsFlagName, "i", []string{}, "Import configuration from the specified files")
	err := importCmd.MarkFlagRequired(importFileLocationsFlagName)
	if err != nil {
		panic(err)
	}
	err = importCmd.MarkFlagFilename(importFileLocationsFlagName, "yaml", "yml", "json")
	if err != nil {
		panic(err)
	}
	return importCmd
}
