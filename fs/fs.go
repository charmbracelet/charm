// Package fs provides an fs.FS implementation for encrypted Charm Cloud storage.
package fs

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
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

var (
	// ErrIsDir is returned when trying to read a directory.
	ErrIsDir = errors.New("is a directory")
	// ErrNotDir is returned when trying to read a file that is not a directory.
	ErrNotDir = errors.New("not a directory")
)

// FS is an implementation of fs.FS, fs.ReadFileFS and fs.ReadDirFS with
// additional write methods. Data is stored across the network on a Charm Cloud
// server, with encryption and decryption happening client-side.
type FS struct {
	cc    *client.Client
	crypt *crypt.Crypt
}

// File implements the fs.File interface.
type File struct {
	Data io.ReadCloser
	Info fs.FileInfo
}

// FileInfo implements the fs.FileInfo interface.
type FileInfo struct {
	charm.FileInfo
	sys *sysFuture
}

type readDirFileFS interface {
	fs.ReadDirFS
	fs.ReadFileFS
}

type sysFuture struct {
	fs   readDirFileFS
	path string
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
	return &FS{cc: cc, crypt: crypt}, nil
}

// Stat returns an fs.FileInfo that describes the file. Implements fs.StatFS.
func (cfs *FS) Stat(name string) (fs.FileInfo, error) {
	info := &FileInfo{
		sys: &sysFuture{
			fs:   cfs,
			path: name,
		},
	}
	ep, err := cfs.EncryptPath(name)
	if err != nil {
		return nil, pathError("stat", name, err)
	}
	p := fmt.Sprintf("/v1/fs/%s", ep)
	resp, err := cfs.cc.AuthedRawRequest("HEAD", p)
	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return nil, fs.ErrNotExist
	} else if err != nil {
		return nil, pathError("stat", name, err)
	}
	defer resp.Body.Close() // nolint:errcheck
	eName := path.Base(ep)
	if eName == "." {
		eName = ""
	}
	// Error if the header file name doesn't match the expected encrypted name.
	// This is a sanity check to ensure that the server is using the right
	// version.
	fileName := resp.Header.Get("X-Name")
	if fileName != eName {
		return nil, pathError("stat", name, fs.ErrInvalid)
	}
	fileMode := resp.Header.Get("X-File-Mode")
	mode, err := strconv.ParseInt(fileMode, 10, 64)
	if err != nil {
		return nil, pathError("stat", name, err)
	}
	isDir := resp.Header.Get("X-Is-Dir")
	lastModified := resp.Header.Get("X-Last-Modified")
	if lastModified == "" {
		lastModified = resp.Header.Get("Last-Modified")
	}
	modTime, err := time.Parse(http.TimeFormat, lastModified)
	if err != nil {
		return nil, pathError("stat", name, err)
	}
	fileSize := resp.Header.Get("X-Size")
	size, err := strconv.ParseInt(fileSize, 10, 64)
	if err != nil {
		return nil, pathError("stat", name, err)
	}
	info.FileInfo = charm.FileInfo{
		Name:    path.Base(fileName),
		Mode:    fs.FileMode(mode),
		IsDir:   isDir == "true",
		ModTime: modTime,
		Size:    size,
	}
	metadata := resp.Header.Get("X-Metadata")
	if metadata != "" {
		b64, err := base64.StdEncoding.DecodeString(metadata)
		if err != nil {
			return nil, pathError("stat", name, err)
		}
		md, err := cfs.crypt.Decrypt(b64)
		if err == nil {
			var fi charm.FileInfo
			err = gob.NewDecoder(bytes.NewBuffer(md)).Decode(&fi)
			if err != nil && err != io.EOF {
				return nil, pathError("stat", name, err)
			}
			if err != io.EOF {
				info.FileInfo = fi
			}
		}
	}
	return info, nil
}

// Open implements Open for fs.FS.
func (cfs *FS) Open(name string) (fs.File, error) {
	f := &File{}
	info, err := cfs.Stat(name)
	if err != nil {
		return nil, err
	}
	f.Info = info.(*FileInfo)
	return f, nil
}

// ReadFile implements fs.ReadFileFS.
func (cfs *FS) ReadFile(name string) ([]byte, error) {
	info, err := cfs.Stat(name)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, pathError("open", name, ErrIsDir)
	}
	ep, err := cfs.EncryptPath(name)
	if err != nil {
		return nil, pathError("open", name, err)
	}
	p := fmt.Sprintf("/v1/fs/%s", ep)
	resp, err := cfs.cc.AuthedRawRequest("GET", p)
	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return nil, fs.ErrNotExist
	} else if err != nil {
		return nil, pathError("open", name, err)
	}
	defer resp.Body.Close() // nolint:errcheck
	switch resp.Header.Get("Content-Type") {
	case "application/octet-stream":
		b := bytes.NewBuffer(nil)
		dec, err := cfs.crypt.NewDecryptedReader(resp.Body)
		if err != nil {
			return nil, pathError("open", name, err)
		}
		_, err = io.Copy(b, dec)
		if err != nil {
			return nil, err
		}
		return b.Bytes(), nil
	default:
		return nil, pathError("open", name, fmt.Errorf("invalid content-type returned from server"))
	}
}

// ReadDir reads the named directory and returns a list of directory entries.
func (cfs *FS) ReadDir(name string) ([]fs.DirEntry, error) {
	f, err := cfs.Open(name)
	if errors.Is(err, fs.ErrNotExist) {
		return []fs.DirEntry{}, nil
	}
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil {
		return nil, pathError("open", name, err)
	}
	if !info.IsDir() {
		return nil, pathError("open", name, ErrNotDir)
	}
	ep, err := cfs.EncryptPath(name)
	if err != nil {
		return nil, pathError("open", name, err)
	}
	p := fmt.Sprintf("/v1/fs/%s", ep)
	resp, err := cfs.cc.AuthedRawRequest("GET", p)
	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return nil, fs.ErrNotExist
	} else if err != nil {
		return nil, pathError("open", name, err)
	}
	defer resp.Body.Close() // nolint:errcheck
	switch resp.Header.Get("Content-Type") {
	case "application/json":
		dirs := []fs.DirEntry{}
		dir := &charm.FileInfo{}
		if err := json.NewDecoder(resp.Body).Decode(&dir); err != nil {
			return nil, pathError("open", name, err)
		}
		for _, e := range dir.Files {
			n, err := cfs.crypt.DecryptLookupField(e.Name)
			if err != nil {
				return nil, pathError("open", name, err)
			}
			fi := e
			fi.Name = n
			if !e.IsDir && e.Metadata != nil {
				md, err := cfs.crypt.Decrypt(e.Metadata)
				if err == nil {
					var ei charm.FileInfo
					err = gob.NewDecoder(bytes.NewBuffer(md)).Decode(&ei)
					if err != nil && err != io.EOF {
						return nil, pathError("open", name, err)
					}
					if err != io.EOF {
						fi = ei
					}
				}
			}
			dirs = append(dirs, &FileInfo{
				FileInfo: fi,
				sys: &sysFuture{
					fs:   cfs,
					path: path.Join(name, fi.Name),
				},
			})
		}
		return dirs, nil
	default:
		return nil, pathError("open", name, fmt.Errorf("invalid content-type returned from server"))
	}
}

// WriteFile encrypts data from data and stores it on the configured Charm Cloud
// server. The fs.FileMode is retained. If the file is in a directory that
// doesn't exist, it and any needed subdirectories are created.
func (cfs *FS) WriteFile(name string, data []byte, perm fs.FileMode) error {
	src := bytes.NewBuffer(data)
	ebuf := bytes.NewBuffer(nil)
	ep, err := cfs.EncryptPath(name)
	if err != nil {
		return err
	}
	fi := charm.FileInfo{
		Name:    path.Base(name),
		IsDir:   false,
		Size:    int64(len(data)),
		Mode:    perm,
		ModTime: time.Now(),
	}

	md := bytes.NewBuffer(nil)
	if err = gob.NewEncoder(md).Encode(fi); err != nil {
		return err
	}
	mde, err := cfs.crypt.Encrypt(md.Bytes())
	if err != nil {
		return err
	}
	eb, err := cfs.crypt.NewEncryptedWriterWithMetadata(ebuf, mde)
	if err != nil {
		return err
	}
	if _, err := io.Copy(eb, src); err != nil {
		return err
	}
	if err := eb.Close(); err != nil {
		return err
	}
	eb.Close() //nolint:errcheck
	// To calculate the Content Length of a multipart request, we need to split
	// the multipart into header, data body, and boundary footer and then
	// calculate the length of each.
	// http/request cannot set Content-Length for a pipe reader
	// https://go.dev/src/net/http/request.go#L891
	databuf := bytes.NewBuffer(nil)
	w := multipart.NewWriter(databuf)
	if _, err := w.CreateFormFile("data", name); err != nil {
		return err
	}
	headlen := databuf.Len()
	header := make([]byte, headlen)
	if _, err := databuf.Read(header); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	bounlen := databuf.Len()
	boun := make([]byte, bounlen)
	if _, err := databuf.Read(boun); err != nil {
		return err
	}
	// TODO: stream the encrypted data to the server, we need to calculate the
	// content length manually.  That is [multipart header length] + [encrypted
	// data length] + [multipart footer length].
	//
	// headlen is the length of the multipart part header, bounlen is the length of the multipart boundary footer.
	contentLength := int64(headlen) + int64(ebuf.Len()) + int64(bounlen)
	// pipe the multipart request to the server
	rr, rw := io.Pipe()
	defer rr.Close() // nolint:errcheck
	go func() {
		defer rw.Close() // nolint:errcheck

		// write multipart header
		if _, err := rw.Write(header); err != nil {
			log.Printf("WriteFile %s error: %v", name, err)
			return
		}
		// chunk the read data into 64MB chunks
		buf := make([]byte, 1024*1024*64)
		for {
			n, err := ebuf.Read(buf)
			if err != nil {
				break
			}
			if _, err := rw.Write(buf[:n]); err != nil {
				log.Printf("WriteFile %s error: %v", name, err)
				return
			}
		}
		// write multipart boundary
		if _, err := rw.Write(boun); err != nil {
			log.Printf("WriteFile %s error: %v", name, err)
			return
		}
	}()
	// Deprecated: remove mode from request query in favor of sasquatch
	// metadata.
	rp := fmt.Sprintf("/v1/fs/%s?mode=%d", ep, perm)
	headers := http.Header{
		"Content-Type":   {w.FormDataContentType()},
		"Content-Length": {fmt.Sprintf("%d", contentLength)},
		"X-File-Mode":    {fmt.Sprintf("%d", perm)},
	}
	resp, err := cfs.cc.AuthedRequest("POST", rp, headers, rr)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

// Remove deletes a file from the Charm Cloud server.
func (cfs *FS) Remove(name string) error {
	ep, err := cfs.EncryptPath(name)
	if err != nil {
		return err
	}
	path := fmt.Sprintf("/v1/fs/%s", ep)
	resp, err := cfs.cc.AuthedRequest("DELETE", path, nil, nil)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

// MkdirAll creates a directory on the configured Charm Cloud server.
func (cfs *FS) MkdirAll(path string, perm fs.FileMode) error {
	ep, err := cfs.EncryptPath(path)
	if err != nil {
		return err
	}
	if !perm.IsDir() {
		return fmt.Errorf("%q is not a directory", path)
	}
	// Deprecated: remove mode from request query in favor of sasquatch
	// metadata.
	rp := fmt.Sprintf("/v1/fs/%s?mode=%d", ep, perm)
	headers := http.Header{
		"X-File-Mode": {fmt.Sprintf("%d", perm)},
	}
	resp, err := cfs.cc.AuthedRequest("POST", rp, headers, nil)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

// Client returns the underlying *client.Client.
func (cfs *FS) Client() *client.Client {
	return cfs.cc
}

// Stat returns an fs.FileInfo that describes the file.
func (f *File) Stat() (fs.FileInfo, error) {
	return f.Info, nil
}

// Read reads bytes from the file returning number of bytes read or an error.
// The error io.EOF will be returned when there is nothing else to read.
func (f *File) Read(b []byte) (int, error) {
	if f.Data == nil {
		sys := f.Info.Sys().(*sysFuture)
		b, err := sys.fs.ReadFile(sys.path)
		if err != nil {
			return 0, err
		}
		f.Data = io.NopCloser(bytes.NewBuffer(b))
	}
	return f.Data.Read(b)
}

// ReadDir returns the directory entries for the directory file. If needed, the
// directory listing will be resolved from the Charm Cloud server.
func (f *File) ReadDir(n int) ([]fs.DirEntry, error) {
	var dirs []fs.DirEntry
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		return nil, ErrNotDir
	}
	if f.Data == nil {
		sys := f.Info.Sys().(*sysFuture)
		dirs, err = sys.fs.ReadDir(sys.path)
		if err != nil {
			return nil, err
		}
		data := bytes.NewBuffer(nil)
		err = json.NewEncoder(data).Encode(dirs)
		if err != nil {
			return nil, err
		}
		f.Data = io.NopCloser(data)
	} else {
		err = json.NewDecoder(f.Data).Decode(&dirs)
		if err != nil {
			return nil, err
		}
	}
	if n > 0 && n < len(dirs) {
		return dirs[:n], nil
	}
	return dirs, nil
}

// Close closes the underlying file datasource.
func (f *File) Close() error {
	if f.Data == nil {
		return nil
	}
	return f.Data.Close()
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

// EncryptPath returns the encrypted path for a given path.
func (cfs *FS) EncryptPath(p string) (string, error) {
	eps := make([]string, 0)
	p = strings.TrimPrefix(p, "charm:")
	p = path.Clean(p)
	if p == "." || p == "/" {
		p = ""
	}
	ps := strings.Split(p, "/")
	for _, p := range ps {
		ep, err := cfs.crypt.EncryptLookupField(p)
		if err != nil {
			return "", err
		}
		if ep == "" {
			continue
		}
		eps = append(eps, ep)
	}
	return strings.Join(eps, "/"), nil
}

// DecryptPath returns the unencrypted path for a given path.
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

func pathError(op, path string, err error) *fs.PathError {
	return &fs.PathError{
		Op:   op,
		Path: path,
		Err:  err,
	}
}
