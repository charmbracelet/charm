package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"

	cfs "github.com/charmbracelet/charm/fs"
	"github.com/spf13/cobra"
)

var (
	fsCmd = &cobra.Command{
		Use:    "fs",
		Hidden: false,
		Short:  "Use the Charm file system.",
		Long:   formatLong(fmt.Sprintf("Commands to set, get and delete data from your Charm Cloud backed file system.")),
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	fsCatCmd = &cobra.Command{
		Use:    "cat PATH",
		Hidden: false,
		Short:  "Output the content of the file at path.",
		Args:   cobra.ExactArgs(1),
		RunE:   fsCat,
	}

	fsListCmd = &cobra.Command{
		Use:    "ls PATH",
		Hidden: false,
		Short:  "List file or directory at path",
		Args:   cobra.ExactArgs(1),
		RunE:   fsList,
	}

	fsTreeCmd = &cobra.Command{
		Use:    "tree PATH",
		Hidden: false,
		Short:  "Print a file system tree from path.",
		Args:   cobra.ExactArgs(1),
		RunE:   fsTree,
	}
)

func printFileInfo(fi fs.FileInfo) {
	fmt.Printf("%s %d %s %s\n", fi.Mode(), fi.Size(), fi.ModTime().Format("Jan 2 15:04"), fi.Name())
}

func printDir(f fs.ReadDirFile) error {
	des, err := f.ReadDir(0)
	if err != nil {
		return err
	}
	for _, v := range des {
		dfi, err := v.Info()
		if err != nil {
			return err
		}
		printFileInfo(dfi)
	}
	return nil
}

func fsCat(cmd *cobra.Command, args []string) error {
	lsfs, err := cfs.NewFS()
	if err != nil {
		return err
	}
	f, err := lsfs.Open(args[0])
	defer f.Close()
	if err != nil {
		return err
	}
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

func fsList(cmd *cobra.Command, args []string) error {
	lsfs, err := cfs.NewFS()
	if err != nil {
		return err
	}
	f, err := lsfs.Open(args[0])
	defer f.Close()
	if err != nil {
		return err
	}
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

func init() {
	fsCmd.AddCommand(fsCatCmd)
	fsCmd.AddCommand(fsListCmd)
	fsCmd.AddCommand(fsTreeCmd)
}
