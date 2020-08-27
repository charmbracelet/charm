package main

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/charm/ui/common"
	"github.com/spf13/cobra"
)

var (
	backupKeysCmd = &cobra.Command{
		Use:                   "backup-keys",
		Hidden:                false,
		Short:                 "Backup your Charm account keys",
		Long:                  formatLong(fmt.Sprintf("%s your Charm account keys.", common.Keyword("Backup"))),
		Args:                  cobra.NoArgs,
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			fileName := "charm-keys-backup.tar"
			dd, err := charm.DataPath()
			if err != nil {
				return err
			}
			err = createTar(dd, fileName)
			if err != nil {
				return err
			}
			printFormatted(fmt.Sprintf("Done! Saved keys to %s.", common.Code(fileName)))
			return nil
		},
	}
)

func createTar(source string, target string) error {
	tarfile, err := os.Create(target)
	if err != nil {
		return err
	}
	defer tarfile.Close()

	tarball := tar.NewWriter(tarfile)
	defer tarball.Close()

	info, err := os.Stat(source)
	if err != nil {
		return nil
	}

	var baseDir string
	if info.IsDir() {
		baseDir = filepath.Base(source)
	}

	return filepath.Walk(source,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			header, err := tar.FileInfoHeader(info, info.Name())
			if err != nil {
				return err
			}

			if baseDir != "" {
				header.Name = filepath.Join(baseDir, strings.TrimPrefix(path, source))
			}

			if err := tarball.WriteHeader(header); err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()
			_, err = io.Copy(tarball, file)
			return err
		})
}
