package main

import (
	"archive/tar"
	"io"
	"os"
	"path/filepath"

	"github.com/charmbracelet/charm"
	"github.com/spf13/cobra"
)

var (
	importKeysCmd = &cobra.Command{
		Use:                   "import-keys BACKUP.tar",
		Hidden:                false,
		Short:                 "Import previously backed up Charm account keys.",
		Args:                  cobra.ExactArgs(1),
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			dd, err := charm.DataPath()
			if err != nil {
				return err
			}
			untar(args[0], filepath.Dir(dd))
			return nil
		},
	}
)

func untar(tarball, target string) error {
	reader, err := os.Open(tarball)
	if err != nil {
		return err
	}
	defer reader.Close()
	tarReader := tar.NewReader(reader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		path := filepath.Join(target, header.Name)
		info := header.FileInfo()
		if info.IsDir() {
			if err = os.MkdirAll(path, info.Mode()); err != nil {
				return err
			}
			continue
		}

		file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(file, tarReader)
		if err != nil {
			return err
		}
	}
	return nil
}
