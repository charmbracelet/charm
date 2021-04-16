package storage

import (
	"io"
	"io/fs"
	"os"
)

// FileStore is the interface storage backends need to implement to act as a
// the datastore for the Charm Cloud server.
type FileStore interface {
	Get(charmID string, path string) (fs.File, error)
	Put(charmID string, path string, r io.Reader, mode fs.FileMode) error
	Delete(charmID string, path string) error
}

// EnsureDir will create the directory for the provided path on the server
// operating system. New directories will have the execute mode set for any
// level of read permission if execute isn't provided in the fs.FileMode.
func EnsureDir(path string, mode fs.FileMode) error {
	_, err := os.Stat(path)
	dp := addExecPermsForMkDir(mode.Perm())
	if os.IsNotExist(err) {
		return os.MkdirAll(path, dp)
	}
	return err
}

func addExecPermsForMkDir(mode fs.FileMode) fs.FileMode {
	if mode.IsDir() {
		return mode
	}
	op := mode.Perm()
	if op&0400 == 0400 {
		op = op | 0100
	}
	if op&0040 == 0040 {
		op = op | 0010
	}
	if op&0004 == 0004 {
		op = op | 0001
	}
	return mode | op | fs.ModeDir
}
