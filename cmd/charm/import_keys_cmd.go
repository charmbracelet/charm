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
				return confirmImportTUI(args[0], dd).Start()
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

func confirmImportTUI(tarPath, dataPath string) *tea.Program {
	type state int

	const (
		ready state = iota
		confirmed
		cancelling
		success
		fail
	)

	var modelAssertionErr = errors.New("could not perform assertion on model")

	type model struct {
		state state
		yes   bool
		err   error
	}

	type successMsg struct{}
	type errMsg error

	untarCmd := func() tea.Msg {
		if err := untar(tarPath, dataPath); err != nil {
			return errMsg(err)
		}
		return successMsg{}
	}

	init := func() (tea.Model, tea.Cmd) {
		return model{state: ready}, nil
	}

	update := func(msg tea.Msg, mdl tea.Model) (tea.Model, tea.Cmd) {
		m, ok := mdl.(model)
		if !ok {
			return model{err: modelAssertionErr}, tea.Quit
		}

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
					return m, untarCmd
				}
				m.state = cancelling
				return m, tea.Quit
			case "y":
				m.yes = true
				m.state = confirmed
				return m, untarCmd
			default:
				if m.state == ready {
					m.yes = false
					m.state = cancelling
					return m, tea.Quit
				}
			}
		case successMsg:
			m.state = success
			return m, tea.Quit
		case errMsg:
			m.state = fail
			m.err = msg
			return m, tea.Quit
		}
		return m, nil
	}

	view := func(mdl tea.Model) string {
		m, ok := mdl.(model)
		if !ok {
			return modelAssertionErr.Error()
		}

		var s string
		switch m.state {
		case ready:
			s = fmt.Sprintf("Looks like you might have some existing keys in %s\n\nWould you like to overwrite them?\n\n", common.Code(dataPath))
			s += common.YesButtonView(m.yes) + " " + common.NoButtonView(!m.yes)
		case success:
			s += fmt.Sprintf("Done! Key imported to %s", common.Code(dataPath))
		case fail:
			s = m.err.Error()
		case cancelling:
			s = "Ok, we won’t do anything. Bye!"
		}

		return formatLong(s) + "\n\n"
	}

	return tea.NewProgram(init, update, view)
}
