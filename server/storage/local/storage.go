package localstorage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	charmfs "github.com/charmbracelet/charm/fs"
	charm "github.com/charmbracelet/charm/proto"
	"github.com/charmbracelet/charm/server/storage"
)

// LocalFileStore is a FileStore implementation that stores files locally in a
// folder.
type LocalFileStore struct {
	Path string
}

// NewLocalFileStore creates a FileStore locally in the provided path. Files
// will be encrypted client-side and stored as regular file system files and
// folders.
func NewLocalFileStore(path string) (*LocalFileStore, error) {
	err := storage.EnsureDir(path, 0700)
	if err != nil {
		return nil, err
	}
	return &LocalFileStore{path}, nil
}

// Get returns an fs.File for the given Charm ID and path.
func (lfs *LocalFileStore) Get(charmID string, path string) (fs.File, error) {
	fp := filepath.Join(lfs.Path, charmID, path)
	info, err := os.Stat(fp)
	if os.IsNotExist(err) {
		return nil, fs.ErrNotExist
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
		fis := make([]charm.FileInfo, 0)
		for _, v := range rds {
			fi, err := v.Info()
			if err != nil {
				return nil, err
			}
			fin := charm.FileInfo{
				Name:    v.Name(),
				IsDir:   fi.IsDir(),
				Size:    fi.Size(),
				ModTime: fi.ModTime(),
				Mode:    fi.Mode(),
			}
			fis = append(fis, fin)
		}
		dir := charm.FileInfo{
			Name:    info.Name(),
			IsDir:   true,
			Size:    0,
			ModTime: info.ModTime(),
			Mode:    info.Mode(),
			Files:   fis,
		}
		buf := bytes.NewBuffer(nil)
		enc := json.NewEncoder(buf)
		err = enc.Encode(dir)
		if err != nil {
			return nil, err
		}
		return &charmfs.DirFile{
			Buffer:   buf,
			FileInfo: info,
		}, nil
	}
	return f, nil
}

// Put reads from the provided io.Reader and stores the data with the Charm ID
// and path.
func (lfs *LocalFileStore) Put(charmID string, path string, r io.Reader, mode fs.FileMode) error {
	if cpath := filepath.Clean(path); cpath == string(os.PathSeparator) {
		return fmt.Errorf("invalid path specified: %s", cpath)
	}

	fp := filepath.Join(lfs.Path, charmID, path)
	if mode.IsDir() {
		return storage.EnsureDir(fp, mode)
	}
	err := storage.EnsureDir(filepath.Dir(fp), mode)
	if err != nil {
		return err
	}
	f, err := os.Create(fp)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, r)
	if err != nil {
		return err
	}
	if mode != 0 {
		return f.Chmod(mode)
	}
	return nil
}

// Delete deletes the file at the given path for the provided Charm ID.
func (lfs *LocalFileStore) Delete(charmID string, path string) error {
	fp := filepath.Join(lfs.Path, charmID, path)
	err := os.RemoveAll(fp)
	if err != nil {
		return err
	}
	return nil
}
