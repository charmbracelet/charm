package cmd

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/tabwriter"

	cfs "github.com/charmbracelet/charm/fs"
	charm "github.com/charmbracelet/charm/proto"
	"github.com/spf13/cobra"
)

const (
	localPath pathType = iota
	remotePath
)

type pathType int

type localRemotePath struct {
	pathType pathType
	path     string
}

type localRemoteFS struct {
	cfs *cfs.FS
}

var (
	isRecursive bool

	// FSCmd is the cobra.Command to use the Charm file system.
	FSCmd = &cobra.Command{
		Use:    "fs",
		Hidden: false,
		Short:  "Use the Charm file system.",
		Long:   paragraph(fmt.Sprintf("Commands to set, get and delete data from your Charm Cloud backed file system.")),
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	fsCatCmd = &cobra.Command{
		Use:    "cat [charm:]PATH",
		Hidden: false,
		Short:  "Output the content of the file at path.",
		Args:   cobra.ExactArgs(1),
		RunE:   fsCat,
	}

	fsCopyCmd = &cobra.Command{
		Use:    "cp [charm:]PATH [charm:]PATH",
		Hidden: false,
		Short:  "Copy a file, preface source or destination with \"charm:\" to specify a remote path.",
		Args:   cobra.ExactArgs(2),
		RunE:   fsCopy,
	}

	fsRemoveCmd = &cobra.Command{
		Use:    "rm [charm:]PATH",
		Hidden: false,
		Short:  "Remove file or directory at path",
		Args:   cobra.ExactArgs(1),
		RunE:   fsRemove,
	}

	fsMoveCmd = &cobra.Command{
		Use:    "mv [charm:]PATH [charm:]PATH",
		Hidden: false,
		Short:  "Move a file, preface source or destination with \"charm:\" to specify a remote path.",
		Args:   cobra.ExactArgs(2),
		RunE:   fsMove,
	}

	fsListCmd = &cobra.Command{
		Use:    "ls [charm:]PATH",
		Hidden: false,
		Short:  "List file or directory at path",
		Args:   cobra.ExactArgs(1),
		RunE:   fsList,
	}

	fsTreeCmd = &cobra.Command{
		Use:    "tree [charm:]PATH",
		Hidden: false,
		Short:  "Print a file system tree from path.",
		Args:   cobra.ExactArgs(1),
		RunE:   fsTree,
	}
)

func newLocalRemoteFS() (*localRemoteFS, error) {
	ccfs, err := cfs.NewFS()
	if err != nil {
		return nil, err
	}
	return &localRemoteFS{cfs: ccfs}, nil
}

func newLocalRemotePath(rawPath string) localRemotePath {
	var pt pathType
	var p string
	if strings.HasPrefix(rawPath, "charm:") {
		pt = remotePath
		p = rawPath[6:]
	} else {
		pt = localPath
		p = rawPath
	}
	return localRemotePath{
		pathType: pt,
		path:     p,
	}
}

func (lrp *localRemotePath) separator() string {
	switch lrp.pathType {
	case localPath:
		return string(os.PathSeparator)
	default:
		return "/"
	}
}

func (lrfs *localRemoteFS) Open(name string) (fs.File, error) {
	p := newLocalRemotePath(name)
	switch p.pathType {
	case localPath:
		return os.Open(p.path)
	case remotePath:
		return lrfs.cfs.Open(p.path)
	default:
		return nil, fmt.Errorf("invalid path type")
	}
}

func (lrfs *localRemoteFS) ReadDir(name string) ([]fs.DirEntry, error) {
	p := newLocalRemotePath(name)
	switch p.pathType {
	case localPath:
		return os.ReadDir(p.path)
	case remotePath:
		return lrfs.cfs.ReadDir(p.path)
	default:
		return nil, fmt.Errorf("invalid path type")
	}
}

func (lrfs *localRemoteFS) write(name string, src fs.File) error {
	stat, err := src.Stat()
	if err != nil {
		return err
	}
	p := newLocalRemotePath(name)
	switch p.pathType {
	case localPath:
		dir := filepath.Dir(p.path)
		if stat.IsDir() {
			dir = dir + "/"
		}
		err = os.MkdirAll(dir, charm.AddExecPermsForMkDir(stat.Mode()))
		if err != nil {
			return err
		}
		if !stat.IsDir() {
			f, err := os.OpenFile(p.path, os.O_RDWR|os.O_CREATE, stat.Mode().Perm())
			if err != nil {
				return err
			}
			defer f.Close()
			_, err = io.Copy(f, src)
			if err != nil {
				return err
			}
		}
	case remotePath:
		if !stat.IsDir() {
			return lrfs.cfs.WriteFile(p.path, src)
		}
	default:
		return fmt.Errorf("invalid path type")
	}
	return nil
}

func (lrfs *localRemoteFS) copy(srcName string, dstName string, recursive bool) error {
	src, err := lrfs.Open(srcName)
	if err != nil {
		return err
	}
	defer src.Close()
	stat, err := src.Stat()
	if err != nil {
		return err
	}
	if stat.IsDir() && !recursive {
		return fmt.Errorf("recursive copy not specified, omitting directory '%s'", srcName)
	}
	if stat.IsDir() && recursive {
		dp := newLocalRemotePath(dstName)
		dstRoot := filepath.Clean(dstName) + dp.separator()
		sp := newLocalRemotePath(srcName)
		parents := len(strings.Split(filepath.Clean(sp.path), sp.separator())) - 1
		return fs.WalkDir(lrfs, srcName, func(wps string, d fs.DirEntry, err error) error {
			if err != nil {
				fmt.Printf("error walking directory %s: %s", srcName, err)
				return err
			}
			wsrc, err := lrfs.Open(wps)
			if err != nil {
				return err
			}
			defer wsrc.Close()
			wp := newLocalRemotePath(wps)
			wpp := strings.Split(filepath.Clean(wp.path), wp.separator())
			rp := path.Join(wpp[parents:]...)
			return lrfs.write(path.Join(dstRoot, rp), wsrc)
		})
	}
	return lrfs.write(dstName, src)
}

func fsCat(cmd *cobra.Command, args []string) error {
	lsfs, err := cfs.NewFS()
	if err != nil {
		return err
	}
	f, err := lsfs.Open(args[0])
	if err != nil {
		return err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return err
	}
	if fi.IsDir() {
		fmt.Printf("cat: %s: Is a directory\n", args[0])
	} else {
		io.Copy(os.Stdout, f)
	}
	return nil
}

func fsMove(cmd *cobra.Command, args []string) error {
	if err := fsCopy(cmd, args); err != nil {
		return err
	}
	return fsRemove(cmd, args[:1])
}

func fsRemove(cmd *cobra.Command, args []string) error {
	lsfs, err := cfs.NewFS()
	if err != nil {
		return err
	}
	return lsfs.Remove(args[0])
}

func fsCopy(cmd *cobra.Command, args []string) error {
	lrfs, err := newLocalRemoteFS()
	if err != nil {
		return err
	}

	src := args[0]
	dst := args[1]
	if strings.HasPrefix(src, "charm:") {
		return lrfs.copy(src, dst, isRecursive)
	}

	// `charm fs cp foo charm:` will copy foo to charm:/foo
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !srcInfo.IsDir() && (dst == "charm:" || dst == "charm:/") {
		dst = "charm:/" + filepath.Base(src)
	}

	return lrfs.copy(src, dst, isRecursive)
}

func fsList(cmd *cobra.Command, args []string) error {
	lsfs, err := cfs.NewFS()
	if err != nil {
		return err
	}
	f, err := lsfs.Open(args[0])
	if err != nil {
		return err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return err
	}
	if fi.IsDir() {
		err = printDir(f.(*cfs.File))
		if err != nil {
			return err
		}
	} else {
		printFileInfo(fi)
	}
	return nil
}

func fsTree(cmd *cobra.Command, args []string) error {
	lsfs, err := cfs.NewFS()
	if err != nil {
		return err
	}
	err = fs.WalkDir(lsfs, args[0], func(path string, d fs.DirEntry, err error) error {
		fmt.Println(path)
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func printFileInfo(fi fs.FileInfo) {
	fmt.Printf("%s %d %s %s\n", fi.Mode(), fi.Size(), fi.ModTime().Format("Jan 2 15:04"), fi.Name())
}

func fprintFileInfo(w io.Writer, fi fs.FileInfo) {
	fmt.Fprintf(w, "%s\t%d\t%s\t %s\n", fi.Mode(), fi.Size(), fi.ModTime().Format("Jan _2 15:04"), fi.Name())
}

func printDir(f fs.ReadDirFile) error {
	des, err := f.ReadDir(0)
	if err != nil {
		return err
	}
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 1, 1, ' ', tabwriter.AlignRight)
	for _, v := range des {
		dfi, err := v.Info()
		if err != nil {
			return err
		}
		fprintFileInfo(w, dfi)
	}
	w.Flush()
	return nil
}

func init() {
	fsCopyCmd.Flags().BoolVarP(&isRecursive, "recursive", "r", false, "copy directories recursively")
	fsMoveCmd.Flags().BoolVarP(&isRecursive, "recursive", "r", false, "move directories recursively")

	FSCmd.AddCommand(fsCatCmd)
	FSCmd.AddCommand(fsCopyCmd)
	FSCmd.AddCommand(fsRemoveCmd)
	FSCmd.AddCommand(fsMoveCmd)
	FSCmd.AddCommand(fsListCmd)
	FSCmd.AddCommand(fsTreeCmd)
}
