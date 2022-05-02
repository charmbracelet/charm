//go:build !openbsd && !windows
// +build !openbsd,!windows

package cmd

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"bazil.org/fuse"
	bfs "bazil.org/fuse/fs"
	cfs "github.com/charmbracelet/charm/fs"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
)

// Node in our virtual filesystem.
type Node struct {
	Mount   *Mount
	Path    string
	Items   map[string]*Node
	Archive fs.DirEntry
	//	sync.RWMutex
}

type Mount struct {
	root *Node
	lsfs *cfs.FS
}

func fsMount(cmd *cobra.Command, args []string) error {
	mountpoint := args[0]

	if _, err := os.Stat(mountpoint); err != nil {
		return err
	}
	c, err := fuse.Mount(
		mountpoint,
		fuse.ReadOnly(),
		fuse.FSName("charmfs"),
	)
	if err != nil {
		return err
	}
	m := &Mount{
		root: &Node{
			Items: make(map[string]*Node),
		},
	}

	fmt.Println("Updating index")
	roottree, err := m.updateIndex()
	if err != nil {
		return err
	}
	fmt.Println("Updating index done")

	errServe := make(chan error)
	go func() {
		err = bfs.Serve(c, &roottree)
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

func (m *Mount) node(name string, arc fs.DirEntry) *Node {
	item := m.root

	l := strings.Split(name, string(filepath.Separator))
	for _, s := range l {
		if len(s) == 0 {
			continue
		}
		// fmt.Println("Finding:", s)
		v, ok := item.Items[s]
		if !ok {
			// path := filepath.Join(l[:k+1]...)
			// fmt.Println("Adding to tree:", path)

			v = &Node{
				Mount:   m,
				Path:    name,
				Items:   make(map[string]*Node),
				Archive: arc,
			}
			item.Items[s] = v
		}

		item = v
	}

	return item
}

func (m *Mount) updateIndex() (bfs.Tree, error) {
	roottree := bfs.Tree{}

	var err error
	m.lsfs, err = cfs.NewFS()
	if err != nil {
		return roottree, err
	}
	if err = fs.WalkDir(m.lsfs, "/", func(path string, d fs.DirEntry, err error) error {
		// fmt.Println(path)
		m.node(path, d)

		return nil
	}); err != nil {
		return roottree, err
	}

	for _, arc := range m.root.Items {
		fmt.Println("Adding to root:", arc.Archive.Name())
		roottree.Add(arc.Archive.Name(), arc)
	}

	return roottree, nil
}

// Attr returns this node's filesystem attributes.
func (node *Node) Attr(_ context.Context, a *fuse.Attr) error {
	// fmt.Println("Attr:", node.Path)
	// a.Inode = node.Inode
	a.Mode = node.Archive.Type().Perm()
	if a.Mode == 0 {
		a.Mode = 0600
	}

	a.Uid = uint32(os.Getuid())
	a.Gid = uint32(os.Getgid())

	info, err := node.Archive.Info()
	if err != nil {
		return err
	}
	a.Size = uint64(info.Size())

	if node.Archive.IsDir() {
		a.Mode |= os.ModeDir
	}

	return nil
}

// Lookup is used to stat items.
func (node *Node) Lookup(_ context.Context, name string) (bfs.Node, error) {
	// fmt.Println("Lookup:", name)
	item, ok := node.Items[name]
	if ok {
		return item, nil
	}

	return nil, fuse.ENOENT
}

// ReadDirAll returns all items directly below this node.
func (node *Node) ReadDirAll(_ context.Context) ([]fuse.Dirent, error) {
	// fmt.Println("ReadDirAll:", node.Path)
	entries := []fuse.Dirent{}

	for k, v := range node.Items {
		ent := fuse.Dirent{Name: k}
		if v.Archive.IsDir() {
			ent.Type = fuse.DT_Dir
		} else if v.Archive.Type().IsRegular() {
			ent.Type = fuse.DT_File
		}

		/*
			ent.Type = fuse.DT_Link
		*/

		entries = append(entries, ent)
	}

	return entries, nil
}

// Open opens a file.
func (node *Node) Open(_ context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (bfs.Handle, error) {
	if !req.Flags.IsReadOnly() {
		return nil, fuse.Errno(syscall.EACCES)
	}
	resp.Flags |= fuse.OpenKeepCache
	return node, nil
}

// Readlink returns the target a symlink is pointing to.
func (node *Node) Readlink(_ context.Context, _ *fuse.ReadlinkRequest) (string, error) {
	return "", nil
	//	return node.Archive.PointsTo, nil
}

// ReadAll reads an entire archive's content.
func (node *Node) ReadAll(_ context.Context) ([]byte, error) {
	// fmt.Println("ReadAll:", node.Path)
	f, err := node.Mount.lsfs.Open("/" + node.Path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if fi.IsDir() {
		return nil, fmt.Errorf("cat: %s: Is a directory", node.Archive.Name())
	}

	return io.ReadAll(f)
}
