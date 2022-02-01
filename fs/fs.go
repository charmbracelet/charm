// Package fs provides an fs.FS implementation for encrypted Charm Cloud storage.
package fs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"mime/multipart"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/charm/client"
	"github.com/charmbracelet/charm/crypt"
	charm "github.com/charmbracelet/charm/proto"
)

// ErrFileTooLarge is returned when a file is too large to upload.
type ErrFileTooLarge struct {
	Limit int64
}

func (e ErrFileTooLarge) Error() string {
	return fmt.Sprintf("file too large: %d", e.Limit)
}

// FS is an implementation of fs.FS, fs.ReadFileFS and fs.ReadDirFS with
// additional write methods. Data is stored across the network on a Charm Cloud
// server, with encryption and decryption happening client-side.
type FS struct {
	cc          *client.Client
	crypt       *crypt.Crypt
	maxFileSize int64 // Max file size in bytes
}

// File implements the fs.File interface.
type File struct {
	data io.ReadCloser
	info *FileInfo
}

// FileInfo implements the fs.FileInfo interface.
type FileInfo struct {
	charm.FileInfo
	sys interface{}
}

type sysFuture struct {
	fs   fs.FS
	path string
}

// DirFile is a fs.File that represents a directory entry.
type DirFile struct {
	Buffer   *bytes.Buffer
	FileInfo fs.FileInfo
}

// Stat returns a fs.FileInfo.
func (df *DirFile) Stat() (fs.FileInfo, error) {
	if df.FileInfo == nil {
		return nil, fmt.Errorf("missing file info")
	}
	return df.FileInfo, nil
}

// Read reads from the DirFile and satisfies fs.FS
func (df *DirFile) Read(buf []byte) (int, error) {
	return df.Buffer.Read(buf)
}

// Close is a no-op but satisfies fs.FS
func (df *DirFile) Close() error {
	return nil
}

// NewFS returns an FS with the default configuration.
func NewFS() (*FS, error) {
	cc, err := client.NewClientWithDefaults()
	if err != nil {
		return nil, err
	}
	return NewFSWithClient(cc)
}

// NewFSWithClient returns an FS with a custom *client.Client.
func NewFSWithClient(cc *client.Client) (*FS, error) {
	crypt, err := crypt.NewCrypt()
	if err != nil {
		return nil, err
	}
	return &FS{cc: cc, crypt: crypt, maxFileSize: 1 << 30}, nil
}

// Open implements Open for fs.FS.
func (cfs *FS) Open(name string) (fs.File, error) {
	f := &File{
		info: &FileInfo{},
	}
	ep, err := cfs.EncryptPath(name)
	if err != nil {
		return nil, pathError(name, err)
	}
	p := fmt.Sprintf("/v1/fs/%s", ep)
	resp, err := cfs.cc.AuthedRawRequest("GET", p)
	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return nil, fs.ErrNotExist
	} else if err != nil {
		return nil, pathError(name, err)
	}
	defer resp.Body.Close()

	switch resp.Header.Get("Content-Type") {
	case "application/json":
		dir := &charm.FileInfo{}
		dec := json.NewDecoder(resp.Body)
		err = dec.Decode(&dir)
		if err != nil {
			return nil, pathError(name, err)
		}
		f.info.FileInfo = *dir
		var des []fs.DirEntry
		for _, de := range dir.Files {
			p := fmt.Sprintf("%s/%s", strings.Trim(ep, "/"), de.Name)
			sf := sysFuture{
				fs:   cfs,
				path: p,
			}
			dn, err := cfs.crypt.DecryptLookupField(de.Name)
			if err != nil {
				return nil, pathError(name, err)
			}
			dei := FileInfo{
				FileInfo: de,
				sys:      sf,
			}
			dei.FileInfo.Name = dn
			des = append(des, &dei)
		}
		f.info.sys = des
	case "application/octet-stream":
		f.info.FileInfo.Name = path.Base(name)
		m, err := strconv.ParseUint(resp.Header.Get("X-File-Mode"), 10, 32)
		if err != nil {
			return nil, pathError(name, err)
		}
		f.info.FileInfo.Mode = fs.FileMode(m)
		b := bytes.NewBuffer(nil)
		dec, err := cfs.crypt.NewDecryptedReader(resp.Body)
		if err != nil {
			return nil, pathError(name, err)
		}
		_, err = io.Copy(b, dec)
		if err != nil {
			return nil, err
		}
		modTime, err := time.Parse(http.TimeFormat, resp.Header.Get("Last-Modified"))
		if err != nil {
			return nil, pathError(name, err)
		}
		f.data = io.NopCloser(b)
		f.info.FileInfo.Size = int64(b.Len())
		f.info.FileInfo.ModTime = modTime
		f.info.FileInfo.IsDir = false
	default:
		return nil, pathError(name, fmt.Errorf("invalid content-type returned from server"))
	}
	return f, nil
}

// ReadFile implements fs.ReadFileFS
func (cfs *FS) ReadFile(name string) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	f, err := cfs.Open(name)
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(buf, f)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// WriteFile encrypts data from the src io.Reader and stores it on the
// configured Charm Cloud server. The fs.FileMode is retained. If the file is
// in a directory that doesn't exist, it and any needed subdirectories are
// created.
func (cfs *FS) WriteFile(name string, src fs.File) error {
	info, err := src.Stat()
	if err != nil {
		return err
	}
	if info.Size() > cfs.maxFileSize {
		return ErrFileTooLarge{Limit: cfs.maxFileSize}
	}
	ebuf := bytes.NewBuffer(nil)
	eb, err := cfs.crypt.NewEncryptedWriter(ebuf)
	if err != nil {
		return err
	}
	_, err = io.Copy(eb, src)
	if err != nil {
		return err
	}
	eb.Close()
	buf := bytes.NewBuffer(nil)
	w := multipart.NewWriter(buf)
	fw, err := w.CreateFormFile("data", name)
	if err != nil {
		return err
	}
	_, err = io.Copy(fw, ebuf)
	if err != nil {
		return err
	}
	w.Close()
	ep, err := cfs.EncryptPath(name)
	if err != nil {
		return err
	}
	path := fmt.Sprintf("/v1/fs/%s?mode=%d", ep, info.Mode())
	headers := http.Header{
		"Content-Type": {w.FormDataContentType()},
	}
	_, err = cfs.cc.AuthedRequest("POST", path, headers, buf)
	return err
}

// Remove deletes a file from the Charm Cloud server.
func (cfs *FS) Remove(name string) error {
	ep, err := cfs.EncryptPath(name)
	if err != nil {
		return err
	}
	path := fmt.Sprintf("/v1/fs/%s", ep)
	_, err = cfs.cc.AuthedRequest("DELETE", path, nil, nil)
	return err
}

// ReadDir reads the named directory and returns a list of directory entries.
func (cfs *FS) ReadDir(name string) ([]fs.DirEntry, error) {
	f, err := cfs.Open(name)
	if err == fs.ErrNotExist {
		return []fs.DirEntry{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return f.(*File).ReadDir(0)
}

// Client returns the underlying *client.Client.
func (cfs *FS) Client() *client.Client {
	return cfs.cc
}

// Stat returns an fs.FileInfo that describes the file.
func (f *File) Stat() (fs.FileInfo, error) {
	return f.info, nil
}

// Read reads bytes from the file returning number of bytes read or an error.
// The error io.EOF will be returned when there is nothing else to read.
func (f *File) Read(b []byte) (int, error) {
	return f.data.Read(b)
}

// ReadDir returns the directory entries for the directory file. If needed, the
// directory listing will be resolved from the Charm Cloud server.
func (f *File) ReadDir(n int) ([]fs.DirEntry, error) {
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf("file is not a directory")
	}
	sys := fi.Sys()
	if sys == nil {
		return nil, fmt.Errorf("missing underlying directory data")
	}
	var des []fs.DirEntry
	switch v := sys.(type) {
	case sysFuture:
		des, err = v.resolve()
		if err != nil {
			return nil, err
		}
		f.info.sys = des
	case []fs.DirEntry:
		des = v
	default:
		return nil, fmt.Errorf("invalid FileInfo sys type")
	}
	if n > 0 && n < len(des) {
		return des[:n], nil
	}
	return des, nil
}

// Close closes the underlying file datasource.
func (f *File) Close() error {
	// directories won't have data
	if f.data == nil {
		return nil
	}
	return f.data.Close()
}

// Name returns the file name.
func (fi *FileInfo) Name() string {
	return fi.FileInfo.Name
}

// Size returns the file size in bytes.
func (fi *FileInfo) Size() int64 {
	return fi.FileInfo.Size
}

// Mode returns the fs.FileMode.
func (fi *FileInfo) Mode() fs.FileMode {
	return fi.FileInfo.Mode
}

// IsDir returns a bool set to true if the file is a directory.
func (fi *FileInfo) IsDir() bool {
	return fi.FileInfo.IsDir
}

// ModTime returns the last modification time for the file.
func (fi *FileInfo) ModTime() time.Time {
	return fi.FileInfo.ModTime
}

// Sys returns the underlying system implementation, may be nil.
func (fi *FileInfo) Sys() interface{} {
	return fi.sys
}

// Type returns the type bits from the fs.FileMode.
func (fi *FileInfo) Type() fs.FileMode {
	return fi.Mode().Type()
}

// Info returns the fs.FileInfo, used to satisfy fs.DirEntry.
func (fi *FileInfo) Info() (fs.FileInfo, error) {
	return fi, nil
}

func (cfs *FS) EncryptPath(path string) (string, error) {
	eps := make([]string, 0)
	path = strings.TrimPrefix(path, "charm:")
	ps := strings.Split(path, "/")
	for _, p := range ps {
		ep, err := cfs.crypt.EncryptLookupField(p)
		if err != nil {
			return "", err
		}
		eps = append(eps, ep)
	}
	return strings.Join(eps, "/"), nil
}

func (cfs *FS) DecryptPath(path string) (string, error) {
	dps := make([]string, 0)
	ps := strings.Split(path, "/")
	for _, p := range ps {
		dp, err := cfs.crypt.DecryptLookupField(p)
		if err != nil {
			return "", err
		}
		dps = append(dps, dp)
	}
	return strings.Join(dps, "/"), nil
}

func (sf sysFuture) resolve() ([]fs.DirEntry, error) {
	f, err := sf.fs.Open(sf.path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	sys := fi.Sys()
	if sys == nil {
		return nil, fmt.Errorf("missing dir entry results")
	}
	return sys.([]fs.DirEntry), nil
}

func pathError(path string, err error) *fs.PathError {
	return &fs.PathError{
		Op:   "open",
		Path: path,
		Err:  err,
	}
}
