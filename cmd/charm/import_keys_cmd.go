package main

import (
	"archive/tar"
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/charm"
	"github.com/spf13/cobra"
)

var (
	importKeysCmd = &cobra.Command{
		Use:                   "import-keys BACKUP.tar",
		Hidden:                false,
		Short:                 "Import previously backed up Charm account keys.",
		Args:                  cobra.ExactArgs(1),
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			dd, err := charm.DataPath()
			if err != nil {
				return err
			}

			empty, err := isEmpty(dd)
			if err != nil {
				return err
			}
			if !empty {
				reader := bufio.NewReader(os.Stdin)
				fmt.Printf("Looks like you might have some existing keys in %s, would you like to overwrite them?\n(yes/no)\n", dd)
				ans, _ := reader.ReadString('\n')
				if strings.ToLower(ans) != "yes\n" {
					fmt.Println("Ok, we won't do anything. Bye!")
					return nil
				}
			}
			err = untar(args[0], filepath.Dir(dd))
			if err != nil {
				return err
			}
			fmt.Printf("Done! Keys imported to %s\n", dd)
			return nil
		},
	}
)

func isEmpty(name string) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}

func untar(tarball, target string) error {
	reader, err := os.Open(tarball)
	if err != nil {
		return err
	}
	defer reader.Close()
	tarReader := tar.NewReader(reader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		path := filepath.Join(target, header.Name)
		info := header.FileInfo()
		if info.IsDir() {
			if err = os.MkdirAll(path, info.Mode()); err != nil {
				return err
			}
			continue
		}

		file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(file, tarReader)
		if err != nil {
			return err
		}
	}
	return nil
}
