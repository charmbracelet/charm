package localstorage

import (
	"bytes"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
)

func TestPut(t *testing.T) {
	tdir := t.TempDir()
	charmID := uuid.New().String()
	buf := bytes.NewBufferString("")
	lfs, err := NewLocalFileStore(tdir)
	if err != nil {
		t.Fatal(err)
	}

	paths := []string{filepath.Join(string(os.PathSeparator), ""), filepath.Join(string(os.PathSeparator), "//")}
	for _, path := range paths {
		err = lfs.Put(charmID, path, buf, fs.FileMode(0o644))
		if err == nil {
			t.Fatalf("expected error when file path is %s", path)
		}

	}

	content := "hello world"
	path := filepath.Join(string(os.PathSeparator), "hello.txt")
	t.Run(path, func(t *testing.T) {
		buf = bytes.NewBufferString(content)
		err = lfs.Put(charmID, path, buf, fs.FileMode(0o644))
		if err != nil {
			t.Fatalf("expected no error when file path is %s, %v", path, err)
		}

		file, err := os.Open(filepath.Join(tdir, charmID, path))
		if err != nil {
			t.Fatalf("expected no error when opening file %s", path)
		}
		defer file.Close() //nolint:errcheck

		fileInfo, err := file.Stat()
		if err != nil {
			t.Fatalf("expected no error when getting file info for %s", path)
		}

		if fileInfo.IsDir() {
			t.Fatalf("expected file %s to be a regular file", path)
		}

		read, err := io.ReadAll(file)
		if err != nil {
			t.Fatalf("expected no error when reading file %s", path)
		}
		if string(read) != content {
			t.Fatalf("expected content to be %s, got %s", content, string(read))
		}
	})

	content = "bar"
	path = filepath.Join(string(os.PathSeparator), "foo", "hello.txt")
	t.Run(path, func(t *testing.T) {
		buf = bytes.NewBufferString(content)
		err = lfs.Put(charmID, path, buf, fs.FileMode(0o644))
		if err != nil {
			t.Fatalf("expected no error when file path is %s, %v", path, err)
		}

		file, err := os.Open(filepath.Join(tdir, charmID, path))
		if err != nil {
			t.Fatalf("expected no error when opening file %s", path)
		}
		defer file.Close() //nolint:errcheck

		fileInfo, err := file.Stat()
		if err != nil {
			t.Fatalf("expected no error when getting file info for %s", path)
		}

		if fileInfo.IsDir() {
			t.Fatalf("expected file %s to be a regular file", path)
		}

		read, err := io.ReadAll(file)
		if err != nil {
			t.Fatalf("expected no error when reading file %s", path)
		}
		if string(read) != content {
			t.Fatalf("expected content to be %s, got %s", content, string(read))
		}
	})
}
