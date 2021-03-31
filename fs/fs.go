package fs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/charmbracelet/charm/client"
	charm "github.com/charmbracelet/charm/proto"
)

type FS struct {
	cc *client.Client
}

type File struct {
	data io.ReadCloser
	info *FileInfo
}

type FileInfo struct {
	charm.FileInfo
	sys interface{}
}

type sysFuture struct {
	fs   fs.FS
	path string
}

func NewFS() (*FS, error) {
	cfg, err := client.ConfigFromEnv()
	if err != nil {
		return nil, err
	}
	cc, err := client.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return NewFSWithClient(cc)
}

func NewFSWithClient(cc *client.Client) (*FS, error) {
	return &FS{cc: cc}, nil
}

func (cfs *FS) Open(name string) (fs.File, error) {
	f := &File{}
	fi := &FileInfo{}
	fi.FileInfo.Name = path.Base(name)
	p := fmt.Sprintf("/v1/fs/%s", name)
	resp, err := cfs.cc.AuthedRawRequest("GET", p)
	if err != nil {
		return nil, err
	}
	switch resp.Header.Get("Content-Type") {
	case "application/json":
		fis := make([]*FileInfo, 0)
		dec := json.NewDecoder(resp.Body)
		err = dec.Decode(&fis)
		if err != nil {
			return nil, err
		}
		fi.FileInfo.IsDir = true
		var des []fs.DirEntry
		for _, de := range fis {
			p := fmt.Sprintf("%s/%s", strings.Trim(name, "/"), de.Name())
			sf := sysFuture{
				fs:   cfs,
				path: p,
			}
			de.sys = sf
			des = append(des, de)
		}
		fi.sys = des
		f.info = fi
	case "application/octet-stream":
		fi.FileInfo.Size = resp.ContentLength
		f.data = resp.Body
		f.info = fi
	default:
		return nil, fmt.Errorf("invalid content-type returned from server")
	}
	return f, nil
}

// TODO this satisfies OsFS in donut, but we probably don't want it here
func (cfs *FS) OpenFile(name string, flag int, perm os.FileMode) (*os.File, error) {
	return nil, fmt.Errorf("not implemented")
}

// TODO this satisfies OsFS in donut, but we probably don't want it here
func (cfs *FS) MkdirAll(path string, perm os.FileMode) error {
	return fmt.Errorf("not implemented")
}

// TODO this satisfies OsFS in donut, but we probably don't want it here
func (cfs *FS) Stat(name string) (os.FileInfo, error) {
	return nil, fmt.Errorf("not implemented")
}

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

// TODO we probably don't need os.FileMode here
func (cfs *FS) WriteFile(name string, data []byte, perm os.FileMode) error {
	buf := bytes.NewBuffer(nil)
	w := multipart.NewWriter(buf)
	fw, err := w.CreateFormFile("data", name)
	if err != nil {
		return err
	}
	dbuf := bytes.NewBuffer(data)
	_, err = io.Copy(fw, dbuf)
	if err != nil {
		return err
	}
	w.Close()
	cfg := cfs.cc.Config
	path := fmt.Sprintf("%s://%s:%d/v1/fs/%s", cfg.HTTPScheme, cfg.Host, cfg.HTTPPort, name)
	req, err := http.NewRequest("POST", path, buf)
	if err != nil {
		return err
	}
	jwt, err := cfs.cc.JWT()
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", fmt.Sprintf("bearer %s", jwt))
	req.Header.Set("Content-Type", w.FormDataContentType())
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server error: %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	return nil
}

func (cfs *FS) Remove(name string) error {
	cfg := cfs.cc.Config
	path := fmt.Sprintf("%s://%s:%d/v1/fs/%s", cfg.HTTPScheme, cfg.Host, cfg.HTTPPort, name)
	req, err := http.NewRequest("DELETE", path, nil)
	if err != nil {
		return err
	}
	jwt, err := cfs.cc.JWT()
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", fmt.Sprintf("bearer %s", jwt))
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server error: %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	return nil
}

func (cfs *FS) ReadDir(name string) ([]fs.DirEntry, error) {
	f, err := cfs.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return f.(*File).ReadDir(0)
}

func (f *File) Stat() (fs.FileInfo, error) {
	return f.info, nil
}

func (f *File) Read(b []byte) (int, error) {
	return f.data.Read(b)
}

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

func (f *File) Close() error {
	// directories won't have data
	if f.data == nil {
		return nil
	}
	return f.data.Close()
}

func (fi *FileInfo) Name() string {
	return fi.FileInfo.Name
}

func (fi *FileInfo) Size() int64 {
	return fi.FileInfo.Size
}

func (fi *FileInfo) Mode() fs.FileMode {
	return fi.FileInfo.Mode
}

func (fi *FileInfo) IsDir() bool {
	return fi.FileInfo.IsDir
}

func (fi *FileInfo) ModTime() time.Time {
	return fi.FileInfo.ModTime
}

func (fi *FileInfo) Sys() interface{} {
	return fi.sys
}

func (fi *FileInfo) Type() fs.FileMode {
	return fi.Mode().Type()
}

func (fi *FileInfo) Info() (fs.FileInfo, error) {
	return fi, nil
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
