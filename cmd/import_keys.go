package cmd

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/charm/client"
	"github.com/charmbracelet/charm/ui/common"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

type (
	confirmationState      int
	confirmationSuccessMsg struct{}
	confirmationErrMsg     struct{ error }
)

const (
	ready confirmationState = iota
	confirmed
	cancelling
	success
	fail
)

var (
	forceImportOverwrite bool

	// ImportKeysCmd is the cobra.Command to import a user's ssh key backup as creaed by `backup-keys`.
	ImportKeysCmd = &cobra.Command{
		Use:                   "import-keys BACKUP.tar",
		Hidden:                false,
		Short:                 "Import previously backed up Charm account keys.",
		Long:                  paragraph(fmt.Sprintf("%s previously backed up Charm account keys.", keyword("Import"))),
		Args:                  cobra.MaximumNArgs(1),
		DisableFlagsInUseLine: false,
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

			if err := os.MkdirAll(dd, 0o700); err != nil {
				return err
			}

			empty, err := isEmpty(dd)
			if err != nil {
				return err
			}

			path := "-"
			if len(args) > 0 {
				path = args[0]
			}
			if !empty && !forceImportOverwrite {
				if common.IsTTY() {
					p := newImportConfirmationTUI(cmd.InOrStdin(), path, dd)
					if _, err := p.Run(); err != nil {
						return err
					}
					return nil
				}
				return fmt.Errorf("not overwriting the existing keys in %s; to force, use -f", dd)
			}

			if isStdin(path) {
				if err := restoreFromReader(cmd.InOrStdin(), dd); err != nil {
					return err
				}
			} else {
				if err := untar(path, dd); err != nil {
					return err
				}
			}

			paragraph(fmt.Sprintf("Done! Keys imported to %s", code(dd)))
			return nil
		},
	}
)

func isStdin(path string) bool {
	fi, _ := os.Stdin.Stat()
	return (fi.Mode()&os.ModeNamedPipe) != 0 || path == "-"
}

func restoreCmd(r io.Reader, path, dataPath string) tea.Cmd {
	return func() tea.Msg {
		if isStdin(path) {
			if err := restoreFromReader(r, dataPath); err != nil {
				return confirmationErrMsg{err}
			}
			return confirmationSuccessMsg{}
		}

		if err := untar(path, dataPath); err != nil {
			return confirmationErrMsg{err}
		}
		return confirmationSuccessMsg{}
	}
}

type confirmationTUI struct {
	reader         io.Reader
	state          confirmationState
	yes            bool
	err            error
	path, dataPath string
}

func (m confirmationTUI) Init() tea.Cmd {
	return nil
}

func (m confirmationTUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.state = cancelling
			return m, tea.Quit
		case "left", "h":
			m.yes = !m.yes
		case "right", "l":
			m.yes = !m.yes
		case "enter":
			if m.yes {
				m.state = confirmed
				return m, restoreCmd(m.reader, m.path, m.dataPath)
			}
			m.state = cancelling
			return m, tea.Quit
		case "y":
			m.yes = true
			m.state = confirmed
			return m, restoreCmd(m.reader, m.path, m.dataPath)
		default:
			if m.state == ready {
				m.yes = false
				m.state = cancelling
				return m, tea.Quit
			}
		}
	case confirmationSuccessMsg:
		m.state = success
		return m, tea.Quit
	case confirmationErrMsg:
		m.state = fail
		m.err = msg
		return m, tea.Quit
	}
	return m, nil
}

func (m confirmationTUI) View() string {
	var s string
	switch m.state {
	case ready:
		s = fmt.Sprintf("Looks like you might have some existing keys in %s\n\nWould you like to overwrite them?\n\n", code(m.dataPath))
		s += common.YesButtonView(m.yes) + " " + common.NoButtonView(!m.yes)
	case success:
		s += fmt.Sprintf("Done! Key imported to %s", code(m.dataPath))
	case fail:
		s = m.err.Error()
	case cancelling:
		s = "Ok, we won’t do anything. Bye!"
	}

	return paragraph(s) + "\n\n"
}

func isEmpty(name string) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer f.Close() // nolint:errcheck

	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}

func restoreFromReader(r io.Reader, dd string) error {
	bts, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	signer, err := ssh.ParsePrivateKey(bts)
	if err != nil {
		return fmt.Errorf("invalid private key: %w", err)
	}

	if signer.PublicKey().Type() != "ssh-ed25519" {
		return fmt.Errorf("only ed25519 keys are allowed, yours is %s", signer.PublicKey().Type())
	}

	keypath := filepath.Join(dd, "charm_ed25519")
	if err := os.WriteFile(keypath, bts, 0o600); err != nil {
		return err
	}

	return os.WriteFile(
		keypath+".pub",
		ssh.MarshalAuthorizedKey(signer.PublicKey()),
		0o600,
	)
}

func untar(tarball, targetDir string) error {
	reader, err := os.Open(tarball)
	if err != nil {
		return err
	}
	defer reader.Close() // nolint:errcheck
	tarReader := tar.NewReader(reader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		// Files are stored in a 'charm' subdirectory in the tar. Strip off the
		// directory info so we can just place the files at the top level of
		// the given target directory.
		filename := filepath.Base(header.Name)

		// Don't create an empty "charm" directory
		if filename == "charm" {
			continue
		}

		path := filepath.Join(targetDir, filename)
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
		defer file.Close() // nolint:errcheck

		for {
			_, err := io.CopyN(file, tarReader, 1024)
			if err != nil {
				if err == io.EOF {
					break
				}
				return err
			}
		}
	}
	return nil
}

// Import Confirmation TUI

func newImportConfirmationTUI(r io.Reader, tarPath, dataPath string) *tea.Program {
	return tea.NewProgram(confirmationTUI{
		reader:   r,
		state:    ready,
		path:     tarPath,
		dataPath: dataPath,
	})
}

func init() {
	ImportKeysCmd.Flags().BoolVarP(&forceImportOverwrite, "force-overwrite", "f", false, "overwrite if keys exist; don’t prompt for input")
}
