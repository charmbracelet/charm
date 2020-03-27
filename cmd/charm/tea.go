package main

import (
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

	menu = Menu{}
)

// MSG

type GotBioMsg *charm.User

// MODEL

type Model struct {
	client  *charm.Client
	user    *charm.User
	spinner spinner.Model
	menu    MenuModel
	err     error
}

// INIT

func initialize() (tea.Model, tea.Cmd) {
	s := spinner.NewModel()
	s.Type = spinner.Dot
	s.ForegroundColor = "244"

	m := Model{
		client:  newCharmClient(),
		spinner: s,
		menu:    MenuModel{},
	}
	return m, getBio
}

// UPDATE

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
			m.menu, _ = menu.Update(msg, m.menu)
			return m, nil
		}

	case GotBioMsg:
		m.user = msg
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = spinner.Update(msg, m.spinner)
		return m, cmd

	default:
		return m, nil
	}
}

// VIEW

func view(model tea.Model) string {
	m, ok := model.(Model)
	if !ok {
		m.err = tea.ModelAssertionErr
	}

	if m.err != nil {
		return errorView(m.err)
	}

	s := charmLogoView()
	if m.user == nil {
		s += spinner.View(m.spinner) + " Fetching your information...\n"
	} else {
		s += bioView(*m.user)
	}

	s += menu.View(m.menu)

	return pad(s)
}

func charmLogoView() string {
	return "\n" + fgBg(" Charm ", cream, purpleBg).Bold().String() + "\n\n"
}

func bioView(u charm.User) string {
	bar := fg("│ ", "241").String()
	username := fg("not set", "214").String()
	if u.Name != "" {
		username = fg(u.Name, purpleFg).String()
	}
	return bar + "Charm ID " + fg(u.CharmID, purpleFg).String() + "\n" +
		bar + "Username " + username
}

func errorView(err error) string {
	return pad("\n" + fgBg("ERROR", "230", "203").String() + " " + err.Error())
}

// SUBSCRIPTIONS

func subscriptions(model tea.Model) tea.Subs {
	m, ok := model.(Model)
	if !ok {
		// TODO: is there a more graceful way to handle this?
		log.Fatal("could not corerce model in main subscriptions function")
	}

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

// HELPERS

func fg(s string, fgColor string) termenv.Style {
	return termenv.String(s).
		Foreground(color(fgColor))
}

func fgBg(s, fgColor, bgColor string) termenv.Style {
	return termenv.String(s).
		Foreground(color(fgColor)).
		Background(color(bgColor))
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
