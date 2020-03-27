package main

import (
	"flag"
	"log"
	"strings"

	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/tea"
	"github.com/charmbracelet/teaparty/spinner"
	"github.com/muesli/termenv"
)

var (
	color    = termenv.ColorProfile().Color
	purpleBg = "#5A56E0"
	purpleFg = "#7571F9"
	cream    = "#FFFDF5"
)

type GotBioMsg *charm.User

type Model struct {
	client  *charm.Client
	user    *charm.User
	spinner spinner.Model
	err     error
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
	s := spinner.NewModel()
	s.Type = spinner.Dot
	s.ForegroundColor = "244"

	m := Model{
		client:  newCharmClient(),
		spinner: s,
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

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = spinner.Update(msg, m.spinner)
		return m, cmd

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

	s := charmLogoView()
	if m.user == nil {
		s += spinner.View(m.spinner) + " Fetching your information...\n"
	} else {
		s += bioView(*m.user)
	}

	return pad(s)
}

func pad(s string) string {
	var r string
	for _, v := range strings.Split(s, "\n") {
		if v == "" {
			r += "\n"
		} else {
			r += "  " + v + "\n"
		}
	}
	return r
}

func charmLogoView() string {
	return "\n" + fgBg(" Charm ", cream, purpleBg) + "\n\n"
}

func bioView(u charm.User) string {
	bar := fg("│ ", "241")
	username := fg("not set", "214")
	if u.Name != "" {
		username = fg(u.Name, purpleFg)
	}
	return bar + "Charm ID " + fg(u.CharmID, purpleFg) + "\n" +
		bar + "Username " + username
}

func fg(s string, fgColor string) string {
	return termenv.String(s).
		Foreground(color(fgColor)).
		String()
}

func fgBg(s, fgColor, bgColor string) string {
	return termenv.String(s).
		Foreground(color(fgColor)).
		Background(color(bgColor)).
		String()
}

func subscriptions(model tea.Model) tea.Subs {
	// TODO: check for error
	m, _ := model.(Model)

	return tea.Subs{
		"tick": func(model tea.Model) tea.Msg {
			return spinner.Sub(m.spinner)
		},
	}

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
