package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"

	charm "github.com/charmbracelet/charm/proto"
)

var ErrFileNotFound = errors.New("file not found")

type FileStore interface {
	Get(charmID string, path string) (io.ReadSeekCloser, error)
	Put(charmID string, path string, r io.Reader) error
	Delete(charmID string, path string) error
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

func (lfs *LocalFileStore) Get(charmID string, path string) (io.ReadSeekCloser, error) {
	fp := filepath.Join(lfs.Path, charmID, path)
	info, err := os.Stat(fp)
	if os.IsNotExist(err) {
		return nil, ErrFileNotFound
	}
	if err != nil {
		return nil, err
	}
	f, err := os.Open(fp)
	if err != nil {
		return nil, err
	}
	// write a directory listing if path is a dir
	if info.IsDir() {
		rds, err := f.ReadDir(0)
		if err != nil {
			return nil, err
		}
		fis := make([]*charm.FileInfo, 0)
		for _, v := range rds {
			fi, err := v.Info()
			if err != nil {
				return nil, err
			}
			fin := &charm.FileInfo{
				Name:    v.Name(),
				IsDir:   v.IsDir(),
				Size:    fi.Size(),
				ModTime: fi.ModTime(),
				Mode:    fi.Mode(),
			}
			fis = append(fis, fin)
		}
		buf := bytes.NewBuffer(nil)
		enc := json.NewEncoder(buf)
		err = enc.Encode(fis)
		if err != nil {
			return nil, err
		}
		return &dirBuffer{bytes.NewReader(buf.Bytes())}, nil
	}
	return f, nil
}

func (lfs *LocalFileStore) Put(charmID string, path string, r io.Reader) error {
	fp := filepath.Join(lfs.Path, charmID, path)
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

func (lfs *LocalFileStore) Delete(charmID string, path string) error {
	fp := filepath.Join(lfs.Path, charmID, path)
	err := os.RemoveAll(fp)
	if err != nil {
		return err
	}
	return nil
}

type dirBuffer struct {
	*bytes.Reader
}

func (db *dirBuffer) Close() error {
	return nil
}

func EnsureDir(path string) error {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return os.MkdirAll(path, 0700)
	}
	return err
}
