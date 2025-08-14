package cmd

import (
	"fmt"
	"io/fs"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"

	"github.com/minio/madmin-go/v4"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/spf13/cobra"
	"github.com/yardenshoham/minio-config-cli/pkg/reconciliation"
)

func newImportCmd() *cobra.Command {
	var importFileLocations []string
	var dryRun bool
	var importCmd = &cobra.Command{
		Use:     "import MINIO_URL ACCESS_KEY SECRET_KEY",
		Short:   "Import configuration from the specified files",
		Example: "minio-config-cli import http://localhost:9000 minioadmin minioadmin --import-file-location=config.yaml",
		Args:    cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			url, err := url.Parse(args[0])
			if err != nil {
				return fmt.Errorf("failed to parse minio url: %w", err)
			}
			secure := url.Scheme == "https"
			creds := credentials.NewStaticV4(args[1], args[2], "")
			madminClient, err := madmin.NewWithOptions(url.Host, &madmin.Options{
				Secure: secure,
				Creds:  creds,
			})
			if err != nil {
				return fmt.Errorf("failed to create madmin client: %w", err)
			}
			minioClient, err := minio.New(url.Host, &minio.Options{
				Creds:  creds,
				Secure: secure,
			})
			if err != nil {
				return fmt.Errorf("failed to create minio client: %w", err)
			}
			logger := slog.New(slog.NewTextHandler(cmd.OutOrStdout(), nil))
			if dryRun {
				logger.Info("running in dry-run mode")
				logger = logger.With("dry-run", "true")
			}
			filePaths := []string{}
			for _, importFileLocation := range importFileLocations {
				err := filepath.WalkDir(importFileLocation, func(path string, d fs.DirEntry, err error) error {
					if err != nil {
						return err
					}
					if d.Type().IsRegular() {
						filePaths = append(filePaths, path)
					}
					return nil
				})
				if err != nil {
					return fmt.Errorf("failed to walk import file locations: %w", err)
				}
			}
			ctx := cmd.Context()
			for _, path := range filePaths {
				file, err := os.Open(path)
				if err != nil {
					return fmt.Errorf("failed to open file %s: %w", path, err)
				}
				defer file.Close()
				config, err := reconciliation.LoadConfig(file)
				if err != nil {
					return fmt.Errorf("failed to load config from file %s: %w", path, err)
				}
				err = reconciliation.Import(ctx, logger.With("file", path), dryRun, madminClient, minioClient, *config)
				if err != nil {
					return fmt.Errorf("failed to import from file %s: %w", path, err)
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
	importCmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "Don't actually modify resources in the MinIO server")
	return importCmd
}
