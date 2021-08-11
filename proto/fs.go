package proto

import (
	"bytes"
	"fmt"
	"io/fs"
	"time"
)

// FileInfo describes a file and is returned by Stat.
type FileInfo struct {
	Name    string      `json:"name"`
	IsDir   bool        `json:"is_dir"`
	Size    int64       `json:"size"`
	ModTime time.Time   `json:"modtime"`
	Mode    fs.FileMode `json:"mode"`
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

// Add execute permissions to an fs.FileMode to mirror read permissions.
func AddExecPermsForMkDir(mode fs.FileMode) fs.FileMode {
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
