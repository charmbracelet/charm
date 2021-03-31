package proto

import (
	"io/fs"
	"time"
)

type FileInfo struct {
	Name    string      `json:"name"`
	IsDir   bool        `json:"is_dir"`
	Size    int64       `json:"size"`
	ModTime time.Time   `json:"modtime"`
	Mode    fs.FileMode `json:"mode"`
}
