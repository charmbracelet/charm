package proto

import (
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
