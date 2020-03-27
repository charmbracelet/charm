package main

import (
	"flag"
	"log"

	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/tea"
)

type GotBioMsg *charm.User

type Model struct {
	client *charm.Client
	user   *charm.User
	err    error
}

// Create a new Charm client
func newCharmClient() *charm.Client {
	i := flag.String("i", "", "identity file (ssh key) path")
	flag.Parse()

	cfg, err := charm.ConfigFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	if *i != "" {
		cfg.SSHKeyPath = *i
		cfg.ForceKey = true
	}

	cc, err := charm.NewClient(cfg)
	if err == charm.ErrMissingSSHAuth {
		log.Fatal("Missing ssh key. Run `ssh-keygen` to make one or set the `CHARM_SSH_KEY_PATH` env var to your private key path.")
	}
	if err != nil {
		log.Fatal(err)
	}

	return cc
}

func initialize() (tea.Model, tea.Cmd) {
	m := Model{
		client: newCharmClient(),
	}
	return m, getBio
}

func update(msg tea.Msg, model tea.Model) (tea.Model, tea.Cmd) {
	m, ok := model.(Model)
	if !ok {
		m.err = tea.ModelAssertionErr
		return m, nil
	}

	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.Type {

		case tea.KeyCtrlC:
			return m, tea.Quit

		default:
			return m, nil
		}

	case GotBioMsg:
		m.user = msg
		return m, tea.Quit

	default:
		return m, nil
	}
}

func view(model tea.Model) string {
	m, ok := model.(Model)
	if !ok {
		m.err = tea.ModelAssertionErr
	}

	// TODO render error if error

	s := "Charm\n\n"
	if m.user == nil {
		s += "Fetching ur info..."
	} else {
		s += bioView(*m.user)
	}

	return s
}

func bioView(u charm.User) string {
	return "Hi, " + u.Name + ". Your Charm ID number is:\n\n" + u.CharmID
}

func subscriptions(model tea.Model) tea.Subs {
	return nil
}

// COMMANDS

func getBio(model tea.Model) tea.Msg {
	m, ok := model.(Model)
	if !ok {
		return tea.NewErrMsgFromErr(tea.ModelAssertionErr)
	}

	user, err := m.client.Bio()
	if err != nil {
		return tea.NewErrMsgFromErr(err)
	}

	return GotBioMsg(user)
}
