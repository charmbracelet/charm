package server

import (
	"io"
	"os"
	"path/filepath"
)

type FileStore interface {
	Get(charmID string, key string, w io.Writer) error
	Put(charmID string, key string, r io.Reader) error
}

type LocalFileStore struct {
	Path string
}

func NewLocalFileStore(path string) (FileStore, error) {
	err := EnsureDir(path)
	if err != nil {
		return nil, err
	}
	return &LocalFileStore{path}, nil
}

func (lfs *LocalFileStore) Get(charmID string, key string, w io.Writer) error {
	fp := filepath.Join(lfs.Path, charmID, key)
	f, err := os.Open(fp)
	defer f.Close()
	if err != nil {
		return err
	}
	_, err = io.Copy(w, f)
	return err
}

func (lfs *LocalFileStore) Put(charmID string, key string, r io.Reader) error {
	fp := filepath.Join(lfs.Path, charmID, key)
	err := EnsureDir(filepath.Dir(fp))
	if err != nil {
		return err
	}
	f, err := os.Create(fp)
	defer f.Close()
	if err != nil {
		return err
	}
	_, err = io.Copy(f, r)
	return err
}

func EnsureDir(path string) error {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return os.MkdirAll(path, 0700)
	}
	return err
}
