package cmd

import (
	"os"
	"testing"

	"github.com/spf13/cobra"
)

func runCobra() *cobra.Command {
	return BackupKeysCmd
}

func TestBackupKeysCmd(t *testing.T) {
	backupFilePath := "./charm-keys-backup.tar"
	if fileOrDirectoryExists(backupFilePath) {
		// delete file or test will fail
		os.Remove(backupFilePath)
	}
	got := runCobra()
	got.Execute()
	fi, err := os.Stat(backupFilePath)
	if err != nil {
		t.Errorf("error reading length of tar file")
	}
	if fi.Size() <= 1024 {
		t.Errorf("tar file should not be empty")
	}
}
