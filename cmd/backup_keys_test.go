package cmd

import (
	"archive/tar"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/charm/testserver"
)

func TestBackupKeysCmd(t *testing.T) {
	backupFilePath := "./charm-keys-backup.tar"
	_ = os.RemoveAll(backupFilePath)
	_ = testserver.SetupTestServer(t)

	if err := BackupKeysCmd.Execute(); err != nil {
		t.Fatalf("command failed: %s", err)
	}

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

		if name := filepath.Base(h.Name); name != "charm_ed25519" && name != "charm_ed25519.pub" {
			t.Errorf("invalid file name: %q", name)
		}
	}

	if len(paths) != 2 {
		t.Errorf("expected at least 2 files (public and private keys), got %d: %v", len(paths), paths)
	}
}
