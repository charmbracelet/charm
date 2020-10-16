package main

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/charm/ui/common"
	"github.com/spf13/cobra"
)

var (
	forceImportOverwrite bool

	importKeysCmd = &cobra.Command{
		Use:                   "import-keys BACKUP.tar",
		Hidden:                false,
		Short:                 "Import previously backed up Charm account keys.",
		Long:                  formatLong(fmt.Sprintf("%s previously backed up Charm account keys.", common.Keyword("Backup"))),
		Args:                  cobra.ExactArgs(1),
		DisableFlagsInUseLine: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !isTTY() && !forceImportOverwrite {
				return errors.New("not a TTY; for non-interactive mode use -f")
			}

			dd, err := charm.DataPath()
			if err != nil {
				return err
			}

			empty, err := isEmpty(dd)
			if err != nil {
				return err
			}

			if !empty && !forceImportOverwrite {
				return newImportConfirmationTUI(args[0], dd).Start()
			}

			err = untar(args[0], filepath.Dir(dd))
			if err != nil {
				return err
			}
			printFormatted(fmt.Sprintf("Done! Keys imported to %s", common.Code(dd)))
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

// Import Confirmation TUI

func newImportConfirmationTUI(tarPath, dataPath string) *tea.Program {
	return tea.NewProgram(confirmationTUI{
		state:    ready,
		tarPath:  tarPath,
		dataPath: dataPath,
	})
}

type confirmationState int

const (
	ready confirmationState = iota
	confirmed
	cancelling
	success
	fail
)

type confirmationSuccessMsg struct{}
type confirmationErrMsg struct{ error }

func untarCmd(tarPath, dataPath string) tea.Cmd {
	return func() tea.Msg {
		if err := untar(tarPath, dataPath); err != nil {
			return confirmationErrMsg{err}
		}
		return confirmationSuccessMsg{}
	}
}

type confirmationTUI struct {
	state             confirmationState
	yes               bool
	err               error
	tarPath, dataPath string
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
				return m, untarCmd(m.tarPath, m.dataPath)
			}
			m.state = cancelling
			return m, tea.Quit
		case "y":
			m.yes = true
			m.state = confirmed
			return m, untarCmd(m.tarPath, m.dataPath)
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
		s = fmt.Sprintf("Looks like you might have some existing keys in %s\n\nWould you like to overwrite them?\n\n", common.Code(m.dataPath))
		s += common.YesButtonView(m.yes) + " " + common.NoButtonView(!m.yes)
	case success:
		s += fmt.Sprintf("Done! Key imported to %s", common.Code(m.dataPath))
	case fail:
		s = m.err.Error()
	case cancelling:
		s = "Ok, we won’t do anything. Bye!"
	}

	return formatLong(s) + "\n\n"
}
