//go:build !openbsd && !windows
// +build !openbsd,!windows

package cmd

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"bazil.org/fuse"
	bfs "bazil.org/fuse/fs"
	cfs "github.com/charmbracelet/charm/fs"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
)

type Mount struct {
	lsfs  *cfs.FS
	cache bool
}

// Dir in our virtual filesystem.
type Dir struct {
	Mount  *Mount
	Parent *Dir
	Name   string
	Mode   os.FileMode
	Items  map[string]interface{}

	cached bool
}

// File in our virtual filesystem.
type File struct {
	Mount  *Mount
	Parent *Dir
	Name   string

	Mode os.FileMode
	Size uint64

	// number of write-capable handles currently open
	writers uint
	// only valid if writers > 0
	data []byte
}

func (m *Mount) Root() (bfs.Node, error) {
	return &Dir{
		Mount: m,
		Items: make(map[string]interface{}),
	}, nil
}

func (m *Mount) Statfs(ctx context.Context, req *fuse.StatfsRequest, resp *fuse.StatfsResponse) error {
	/*
		Blocks  uint64 // Total data blocks in file system.
		Bfree   uint64 // Free blocks in file system.
		Bavail  uint64 // Free blocks in file system if you're not root.
		Files   uint64 // Total files in file system.
		Ffree   uint64 // Free files in file system.
		Bsize   uint32 // Block size
		Namelen uint32 // Maximum file name length?
		Frsize  uint32 // Fragment size, smallest addressable data size in the file system.
	*/

	//TODO: this is obviously fake data
	resp.Blocks = 1000 * 1000
	resp.Bfree = 1000 * 1000
	resp.Bavail = 1000 * 1000
	resp.Files = 1
	resp.Ffree = 1000
	resp.Bsize = 1024
	resp.Namelen = 256
	resp.Frsize = 1024
	return nil
}

func (d *Dir) Path() string {
	if d.Parent == nil {
		return "/"
	}

	r := "/"
	p := d.Parent
	for p != nil {
		r = filepath.Join(p.Name, r)
		p = p.Parent
	}
	return filepath.Join(r, d.Name)
}

func (f *File) Path() string {
	r := "/"
	p := f.Parent
	for p != nil {
		r = filepath.Join(p.Name, r)
		p = p.Parent
	}
	return filepath.Join(r, f.Name)
}

func (d *Dir) Attr(ctx context.Context, a *fuse.Attr) error {
	// fmt.Println("Attr:", d.Path())
	// a.Inode = node.Inode

	/*
		st, err := fs.Stat(d.Mount.lsfs, d.Path())
		if err != nil {
			return fuse.ENOENT
		}

		// a.Mode = st.Mode()
		a.Size = uint64(st.Size())
	*/

	a.Uid = uint32(os.Getuid())
	a.Gid = uint32(os.Getgid())
	if a.Mode == 0 {
		a.Mode = os.ModeDir | 0700
	}

	return nil
}

// Lookup is used to stat items.
func (d *Dir) Lookup(_ context.Context, name string) (bfs.Node, error) {
	fmt.Println("Lookup:", filepath.Join(d.Path(), name))

	if d.Mount.cache {
		if item, ok := d.Items[name]; ok {
			fmt.Println("Lookup cached!")
			switch item := item.(type) {
			case *Dir:
				return item, nil
			case *File:
				return item, nil
			case nil:
				// cached non-existent
				return nil, fuse.ENOENT
			}
		}
	}

	st, err := fs.Stat(d.Mount.lsfs, filepath.Join(d.Path(), name))
	if err != nil {
		d.Items[name] = nil
		return nil, fuse.ENOENT
	}

	if st.IsDir() {
		return &Dir{
			Mount:  d.Mount,
			Parent: d,
			Name:   name,
			Mode:   st.Mode(),
			Items:  make(map[string]interface{}),
		}, nil
	}

	return &File{
		Mount:  d.Mount,
		Parent: d,
		Name:   name,
		Mode:   st.Mode(),
		Size:   uint64(st.Size()),
	}, nil
}

// ReadDirAll returns all items directly below this node.
func (d *Dir) ReadDirAll(_ context.Context) ([]fuse.Dirent, error) {
	fmt.Println("ReadDirAll:", d.Path())
	entries := []fuse.Dirent{}

	if d.Mount.cache && d.cached {
		fmt.Println("ReadDirAll cached!")
		for name, item := range d.Items {
			switch item.(type) {
			case *Dir:
				entries = append(entries, fuse.Dirent{
					Name: name,
					Type: fuse.DT_Dir,
				})
			case *File:
				entries = append(entries, fuse.Dirent{
					Name: name,
					Type: fuse.DT_File,
				})
			}
		}

		return entries, nil
	}

	d.cached = true
	if err := fs.WalkDir(d.Mount.lsfs, d.Path(), func(path string, de fs.DirEntry, err error) error {
		if path == d.Path() {
			return nil
		}
		fmt.Println("Found:", path, filepath.Base(path))

		ent := fuse.Dirent{Name: filepath.Base(path)}
		if de.IsDir() {
			ent.Type = fuse.DT_Dir

			item := &Dir{
				Mount:  d.Mount,
				Parent: d,
				Name:   filepath.Base(path),
				Mode:   de.Type().Perm(),
				Items:  make(map[string]interface{}),
			}
			d.Items[item.Name] = item
		} else {
			ent.Type = fuse.DT_File

			info, err := de.Info()
			if err != nil {
				return err
			}

			item := &File{
				Mount:  d.Mount,
				Parent: d,
				Name:   filepath.Base(path),
				Mode:   de.Type().Perm(),
				Size:   uint64(info.Size()),
			}
			d.Items[item.Name] = item
		}

		entries = append(entries, ent)
		if de.IsDir() {
			return fs.SkipDir
		}

		return nil
	}); err != nil {
		return entries, err
	}

	return entries, nil
}

// Attr returns this node's filesystem attributes.
func (f *File) Attr(_ context.Context, a *fuse.Attr) error {
	// fmt.Println("Attr:", f.Path())
	// a.Inode = f.Inode

	/*
		st, err := fs.Stat(f.Mount.lsfs, f.Path())
		if err != nil {
			return fuse.ENOENT
		}

		a.Mode = st.Mode()
		a.Size = uint64(st.Size())
	*/

	a.Mode = f.Mode
	a.Size = f.Size

	a.Uid = uint32(os.Getuid())
	a.Gid = uint32(os.Getgid())
	if a.Mode == 0 {
		a.Mode = 0644
	}

	return nil
}

func fsMount(cmd *cobra.Command, args []string) error {
	mountpoint := args[0]

	if _, err := os.Stat(mountpoint); err != nil {
		return err
	}
	c, err := fuse.Mount(
		mountpoint,
		// fuse.ReadOnly(),
		fuse.FSName("charmfs"),
	)
	if err != nil {
		return err
	}
	m := &Mount{
		cache: true,
	}
	m.lsfs, err = cfs.NewFS()
	if err != nil {
		return err
	}

	errServe := make(chan error)
	go func() {
		err = bfs.Serve(c, m)
		if err != nil {
			errServe <- err
		}

		<-c.Ready
		errServe <- c.MountError
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errServe:
		return err

	case <-sigs:
		fmt.Println("\nShutting down...")
		if err := fuse.Unmount(mountpoint); err != nil {
			fmt.Printf("Error umounting: %s\n", err)
		}
		return c.Close()
	}
}

// Open opens a file.
func (f *File) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (bfs.Handle, error) {
	fmt.Printf("Opening %s\n", f.Path())

	if req.Flags.IsReadOnly() {
		// we don't need to track read-only handles
		return f, nil
	}

	if f.writers == 0 {
		// load data
		var err error
		f.data, err = f.ReadAll(ctx)
		if err != nil {
			return nil, err
		}
	}

	f.writers++
	return f, nil
}

type NodeFile struct {
	node *File
	*bytes.Reader
}

func (nf *NodeFile) Stat() (fs.FileInfo, error) {
	return nf, nil
}

func (nf *NodeFile) Name() string {
	return nf.node.Name
}

func (nf *NodeFile) Size() int64 {
	return int64(nf.node.Size)
}

func (nf *NodeFile) Mode() fs.FileMode {
	return nf.node.Mode
}

func (nf *NodeFile) ModTime() time.Time {
	//TODO: implement
	return time.Now()
}

func (nf *NodeFile) IsDir() bool {
	return false
}

func (nf *NodeFile) Sys() interface{} {
	return nil
}

func (nf *NodeFile) Close() error {
	return nil
}

func (d *Dir) Rename(ctx context.Context, req *fuse.RenameRequest, newDir bfs.Node) error {
	f, ok := d.Items[req.OldName]
	if !ok {
		return fuse.ENOENT
	}

	ff := f.(*File)

	dst := filepath.Join(newDir.(*Dir).Path(), req.NewName)
	fmt.Printf("Renaming %s to %s\n", ff.Path(), dst)

	ff.ReadAll(ctx)

	if err := d.Mount.lsfs.WriteFile(dst, &NodeFile{ff, bytes.NewReader(ff.data)}); err != nil {
		return err
	}

	d.Items[req.OldName] = nil

	return d.Mount.lsfs.Remove(ff.Path())
}

func (f *File) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	if req.Flags.IsReadOnly() {
		// we don't need to track read-only handles
		return nil
	}

	fmt.Printf("Releasing %s %d\n", f.Path(), len(f.data))

	f.writers--
	if f.writers == 0 {
		//TODO: uncache?
		//		node.data = nil
	}
	return nil
}

// ReadAll reads an entire archive's content.
func (f *File) ReadAll(_ context.Context) ([]byte, error) {
	fmt.Println("ReadAll:", f.Path())
	if f.Mount.cache && len(f.data) > 0 {
		fmt.Println("ReadAll cached!")
		return f.data, nil
	}

	fm, err := f.Mount.lsfs.Open(f.Path())
	if err != nil {
		return nil, err
	}
	defer fm.Close()

	fi, err := fm.Stat()
	if err != nil {
		return nil, err
	}
	if fi.IsDir() {
		return nil, fmt.Errorf("cat: %s: Is a directory", f.Path())
	}

	f.data, err = io.ReadAll(fm)
	return f.data, err
}

const maxInt = int(^uint(0) >> 1)

func (f *File) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	fmt.Printf("Writing %d bytes to %s\n", len(req.Data), f.Path())
	// expand the buffer if necessary
	newLen := req.Offset + int64(len(req.Data))
	if newLen > int64(maxInt) {
		return fuse.Errno(syscall.EFBIG)
	}

	if newLen := int(newLen); newLen > len(f.data) {
		f.data = append(f.data, make([]byte, newLen-len(f.data))...)
	}

	n := copy(f.data[req.Offset:], req.Data)
	resp.Size = n
	return nil
}

var _ = bfs.HandleFlusher(&File{})

func (f *File) Flush(ctx context.Context, req *fuse.FlushRequest) error {
	fmt.Printf("Flushing %s %d %d\n", f.Path(), len(f.data), f.writers)
	if f.writers == 0 {
		// Read-only handles also get flushes. Make sure we don't
		// overwrite valid file contents with a nil buffer.
		return nil
	}

	if err := f.Mount.lsfs.WriteFile(f.Path(), &NodeFile{f, bytes.NewReader(f.data)}); err != nil {
		return err
	}
	f.Size = uint64(len(f.data))

	return nil
}

var _ = bfs.NodeSetattrer(&File{})

func (f *File) Setattr(ctx context.Context, req *fuse.SetattrRequest, resp *fuse.SetattrResponse) error {
	fmt.Println("Setattr for", f.Path(), req.String())
	if req.Valid.Mode() {
		f.Mode = req.Mode
	}
	if req.Valid.Size() {
		fmt.Printf("Setattr %d bytes for %s\n", req.Size, f.Path())
		if req.Size > uint64(maxInt) {
			return fuse.Errno(syscall.EFBIG)
		}
		newLen := int(req.Size)
		switch {
		case newLen > len(f.data):
			f.data = append(f.data, make([]byte, newLen-len(f.data))...)
		case newLen < len(f.data):
			f.data = f.data[:newLen]
		}
		f.Size = req.Size
	}
	return nil
}

func (f *File) Fsync(ctx context.Context, req *fuse.FsyncRequest) error {
	fmt.Println("Fsync for", f.Path(), req.String())
	return nil
}

var _ = bfs.NodeMkdirer(&Dir{})

func (d *Dir) Mkdir(ctx context.Context, req *fuse.MkdirRequest) (bfs.Node, error) {
	fmt.Printf("Creating dir in %s: %s\n", d.Path(), req.Name)

	dir := &Dir{
		Mount:  d.Mount,
		Parent: d,
		Name:   req.Name,
		Mode:   0700,
		Items:  make(map[string]interface{}),
	}
	d.Items[req.Name] = dir

	return dir, nil
}

var _ = bfs.NodeCreater(&Dir{})

func (d *Dir) Create(ctx context.Context, req *fuse.CreateRequest, resp *fuse.CreateResponse) (bfs.Node, bfs.Handle, error) {
	fmt.Printf("Creating file in %s: %s\n", d.Path(), req.Name)

	f := &File{
		Mount:   d.Mount,
		Parent:  d,
		Name:    req.Name,
		Mode:    0644,
		writers: 1,
	}
	d.Items[req.Name] = f

	return f, f, nil
}

var _ = bfs.NodeRemover(&Dir{})

func (d *Dir) Remove(ctx context.Context, req *fuse.RemoveRequest) error {
	fmt.Printf("Removing file in %s: %s\n", d.Path(), req.Name)

	d.Items[req.Name] = nil

	return d.Mount.lsfs.Remove(filepath.Join(d.Path(), req.Name))
}
