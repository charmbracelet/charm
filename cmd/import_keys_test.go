package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/charm/testserver"
)

func TestImportKeysFromStdin(t *testing.T) {
	c := testserver.SetupTestServer(t)

	var r bytes.Buffer
	BackupKeysCmd.SetArgs([]string{"-o", "-"})
	BackupKeysCmd.SetOut(&r)
	if err := BackupKeysCmd.Execute(); err != nil {
		t.Fatalf(err.Error())
	}

	dd, _ := c.DataPath()
	if err := os.RemoveAll(dd); err != nil {
		t.Fatalf(err.Error())
	}

	ImportKeysCmd.SetIn(&r)
	ImportKeysCmd.SetArgs([]string{"-f", "-"})
	if err := ImportKeysCmd.Execute(); err != nil {
		t.Fatalf(err.Error())
	}

	if _, err := os.Stat(filepath.Join(dd, "charm_ed25519")); err != nil {
		t.Fatalf(err.Error())
	}
	if _, err := os.Stat(filepath.Join(dd, "charm_ed25519.pub")); err != nil {
		t.Fatalf(err.Error())
	}
}

func TestImportKeysFromFile(t *testing.T) {
	c := testserver.SetupTestServer(t)

	f := filepath.Join(t.TempDir(), "backup.tar")

	BackupKeysCmd.SetArgs([]string{"-o", f})
	if err := BackupKeysCmd.Execute(); err != nil {
		t.Fatalf(err.Error())
	}

	dd, _ := c.DataPath()
	if err := os.RemoveAll(dd); err != nil {
		t.Fatalf(err.Error())
	}

	ImportKeysCmd.SetArgs([]string{"-f", f})
	if err := ImportKeysCmd.Execute(); err != nil {
		t.Fatalf(err.Error())
	}

	if _, err := os.Stat(filepath.Join(dd, "charm_ed25519")); err != nil {
		t.Fatalf(err.Error())
	}
	if _, err := os.Stat(filepath.Join(dd, "charm_ed25519.pub")); err != nil {
		t.Fatalf(err.Error())
	}
}
