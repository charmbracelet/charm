package main

import (
	"archive/tar"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

var (
	backupKeysCmd = &cobra.Command{
		Use:    "backup-keys",
		Hidden: false,
		Short:  "Backup your Charm account keys.",
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cc := initCharmClient()
			fileName := "charm-keys-backup.tar"
			backupFile, err := os.Create(fileName)
			if err != nil {
				return err
			}
			err = backupFile.Chmod(0600)
			if err != nil {
				return err
			}
			defer backupFile.Close()
			tarball := tar.NewWriter(backupFile)
			defer tarball.Close()
			for _, kp := range cc.AuthKeyPaths() {
				addFileToTar(tarball, kp)
				addFileToTar(tarball, fmt.Sprintf("%s.pub", kp))
			}
			fmt.Printf("Done! Saved keys to `%s`\n", fileName)
			return nil
		},
	}
)

func addFileToTar(tarball *tar.Writer, fp string) error {
	privKeyInfo, err := os.Stat(fp)
	if err != nil {
		return err
	}
	header, err := tar.FileInfoHeader(privKeyInfo, privKeyInfo.Name())
	if err != nil {
		return err
	}
	err = tarball.WriteHeader(header)
	if err != nil {
		return err
	}
	f, err := os.Open(fp)
	if err != nil {
		return err
	}
	_, err = io.Copy(tarball, f)
	defer f.Close()
	if err != nil {
		return err
	}
	return nil
}
