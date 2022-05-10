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
	"github.com/muesli/sasquatch"
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
	err := storage.EnsureDir(path, 0o700)
	if err != nil {
		return nil, err
	}
	return &LocalFileStore{path}, nil
}

// Stat returns the FileInfo for the given Charm ID and path.
func (lfs *LocalFileStore) Stat(charmID, path string) (fs.FileInfo, error) {
	fp := filepath.Join(lfs.Path, charmID, path)
	f, err := os.Open(fp)
	if os.IsNotExist(err) {
		return nil, fs.ErrNotExist
	}
	if err != nil {
		return nil, err
	}
	i, err := f.Stat()
	if err != nil {
		return nil, err
	}
	var md []byte
	if !i.IsDir() {
		md, err = sasquatch.Metadata(f)
		if err != nil {
			return nil, err
		}
	}
	name := i.Name()
	if name == charmID {
		name = ""
	}
	in := &charmfs.FileInfo{
		FileInfo: charm.FileInfo{
			Name:     name,
			IsDir:    i.IsDir(),
			Size:     i.Size(),
			ModTime:  i.ModTime(),
			Mode:     i.Mode(),
			Metadata: md,
		},
	}
	// Get the actual size of the files in a directory
	if i.IsDir() {
		in.FileInfo.Size = 0
		if err = filepath.Walk(fp, func(path string, info fs.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}
			in.FileInfo.Size += info.Size()
			return nil
		}); err != nil {
			return nil, err
		}
	}
	return in, nil
}

// Get returns an fs.File for the given Charm ID and path.
func (lfs *LocalFileStore) Get(charmID string, path string) (fs.File, error) {
	data := bytes.NewBuffer(nil)
	fp := filepath.Join(lfs.Path, charmID, path)
	info, err := lfs.Stat(charmID, path)
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
			var md []byte
			if !v.IsDir() {
				sf, err := os.Open(filepath.Join(fp, v.Name()))
				if err != nil {
					return nil, err
				}
				md, err = sasquatch.Metadata(sf)
				if err != nil {
					return nil, err
				}
			}
			fin := charm.FileInfo{
				Name:     v.Name(),
				IsDir:    fi.IsDir(),
				Size:     fi.Size(),
				ModTime:  fi.ModTime(),
				Mode:     fi.Mode(),
				Metadata: md,
			}
			fis = append(fis, fin)
		}
		dir := charm.FileInfo{
			Name:    info.Name(),
			IsDir:   info.IsDir(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
			Mode:    info.Mode(),
			Files:   fis,
		}
		if err := json.NewEncoder(data).Encode(dir); err != nil {
			return nil, err
		}
	} else {
		_, err := io.Copy(data, f)
		if err != nil {
			return nil, err
		}
	}
	return &charmfs.File{
		Data: io.NopCloser(data),
		Info: info,
	}, nil
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
	defer f.Close() // nolint:errcheck
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
	return os.RemoveAll(fp)
}
