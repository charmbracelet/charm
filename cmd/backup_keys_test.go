package cmd

import (
	"archive/tar"
	"io"
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

	f, err := os.Open(backupFilePath)
	if err != nil {
		t.Fatalf("error opening tar file: %s", err)
	}
	t.Cleanup(func() {
		_ = f.Close()
	})
	fi, err := f.Stat()
	if err != nil {
		t.Fatalf("error reading length of tar file: %s", err)
	}
	if fi.Size() <= 1024 {
		t.Errorf("tar file should not be empty")
	}

	var paths []string
	r := tar.NewReader(f)
	for {
		h, err := r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Errorf("error opening tar file: %s", err)
		}
		paths = append(paths, h.Name)
	}

	if len(paths) < 2 {
		t.Errorf("expected at least 2 files (public and private keys), got %d: %v", len(paths), paths)
	}
}
