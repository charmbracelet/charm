package charmclient

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/charm"
)

type NewClientMsg *charm.Client

type SSHAuthErrorMsg struct {
	Err error
}

type ErrMsg struct {
	Err error
}

// NewClient is a Bubble Tea command for creating a Charm client
func NewClient(cfg *charm.Config) tea.Cmd {
	return func() tea.Msg {
		cc, err := charm.NewClient(cfg)

		if err == charm.ErrMissingSSHAuth {
			return SSHAuthErrorMsg{err}
		} else if err != nil {
			return ErrMsg{err}
		}

		return NewClientMsg(cc)
	}
}
