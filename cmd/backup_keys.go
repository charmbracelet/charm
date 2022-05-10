package cmd

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/charmbracelet/charm/client"
	"github.com/spf13/cobra"
)

var backupOutputFile string

func init() {
	BackupKeysCmd.Flags().StringVarP(&backupOutputFile, "output", "o", "charm-keys-backup.tar", "keys backup filepath")
}

// BackupKeysCmd is the cobra.Command to back up a user's account SSH keys.
var BackupKeysCmd = &cobra.Command{
	Use:                   "backup-keys",
	Hidden:                false,
	Short:                 "Backup your Charm account keys",
	Long:                  paragraph(fmt.Sprintf("%s your Charm account keys to a tar archive file. \nYou can restore your keys from backup using import-keys. \nRun `charm import-keys -help` to learn more.", keyword("Backup"))),
	Args:                  cobra.NoArgs,
	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := client.ConfigFromEnv()
		if err != nil {
			return err
		}

		cc, err := client.NewClient(cfg)
		if err != nil {
			return err
		}

		dd, err := cc.DataPath()
		if err != nil {
			return err
		}

		if err := validateDirectory(dd); err != nil {
			return err
		}

		backupPath := backupOutputFile
		if backupPath == "-" {
			exp := regexp.MustCompilePOSIX("charm_(rsa|ed25519)$")
			paths, err := getKeyPaths(dd, exp)
			if err != nil {
				return err
			}
			if len(paths) != 1 {
				return fmt.Errorf("backup to stdout only works with 1 key, you have %d", len(paths))
			}
			bts, err := os.ReadFile(paths[0])
			if err != nil {
				return err
			}
			_, _ = fmt.Fprint(cmd.OutOrStdout(), string(bts))
			return nil
		}

		if !strings.HasSuffix(backupPath, ".tar") {
			backupPath = backupPath + ".tar"
		}

		if fileOrDirectoryExists(backupPath) {
			fmt.Printf("Not creating backup file: %s already exists.\n\n", code(backupPath))
			os.Exit(1)
		}

		if err := os.MkdirAll(filepath.Dir(backupPath), 0o754); err != nil {
			return err
		}

		if err := createTar(dd, backupPath); err != nil {
			return err
		}

		fmt.Printf("Done! Saved keys to %s.\n\n", code(backupPath))
		return nil
	},
}

func fileOrDirectoryExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

func validateDirectory(path string) error {
	info, err := os.Stat(path)
	if err == nil {
		if !info.IsDir() {
			return fmt.Errorf("%v is not a directory, but it should be", path)
		}

		files, err := os.ReadDir(path)
		if err != nil {
			return err
		}

		foundKeys := 0
		keyPattern := regexp.MustCompile(`charm_(rsa|ed25519)(\.pub)?`)

		for _, f := range files {
			if !f.IsDir() && keyPattern.MatchString(f.Name()) {
				foundKeys++
			}
		}
		if foundKeys < 2 {
			return fmt.Errorf("we didn’t find any keys to backup in %s", path)
		}

		// Everything looks OK!
		return nil
	} else if os.IsNotExist(err) {
		return fmt.Errorf("'%v' does not exist", path)
	} else {
		return err
	}
}

func createTar(source string, target string) error {
	tarfile, err := os.Create(target)
	if err != nil {
		return err
	}
	defer tarfile.Close() // nolint:errcheck

	tarball := tar.NewWriter(tarfile)
	defer tarball.Close() // nolint:errcheck

	info, err := os.Stat(source)
	if err != nil {
		return nil
	}

	var baseDir string
	if info.IsDir() {
		baseDir = filepath.Base(source)
	}

	exp := regexp.MustCompilePOSIX("charm_(rsa|ed25519)(.pub)?$")

	paths, err := getKeyPaths(source, exp)
	if err != nil {
		return err
	}

	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}

		if baseDir != "" {
			header.Name = filepath.Join(baseDir, strings.TrimPrefix(path, source))
		}

		if err := tarball.WriteHeader(header); err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close() // nolint:errcheck

		if _, err := io.Copy(tarball, file); err != nil {
			return err
		}
		if err := file.Close(); err != nil {
			return err
		}
	}

	if err := tarball.Close(); err != nil {
		return err
	}
	return tarfile.Close()
}

func getKeyPaths(source string, filter *regexp.Regexp) ([]string, error) {
	var result []string
	err := filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if filter.MatchString(path) {
			result = append(result, path)
		}

		return nil
	})
	return result, err
}
