package charmclient

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/charm/client"
	charm "github.com/charmbracelet/charm/proto"
)

// NewClientMsg is sent when we've successfully created a charm client.
type NewClientMsg *client.Client

// SSHAuthErrorMsg is sent when charm client creation has failed due to a
// problem with SSH.
type SSHAuthErrorMsg struct {
	Err error
}

// ErrMsg is sent for general, non-SSH related errors encountered when creating
// a Charm Client.
type ErrMsg struct {
	Err error
}

// NewClient is a Bubble Tea command for creating a Charm client.
func NewClient(cfg *client.Config) tea.Cmd {
	return func() tea.Msg {
		cc, err := client.NewClient(cfg)

		if err == charm.ErrMissingSSHAuth {
			return SSHAuthErrorMsg{err}
		} else if err != nil {
			return ErrMsg{err}
		}

		return NewClientMsg(cc)
	}
}
