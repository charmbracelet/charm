package main

import (
	"github.com/charmbracelet/tea"
)

type Model struct{}

func initialize() (tea.Model, tea.Cmd) {
	m := Model{}
	return m, nil
}

func update(msg tea.Msg, model tea.Model) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			fallthrough
		case "esc":
			fallthrough
		case "ctrl+c":
			return model, tea.Quit
		}

	}

	return model, nil
}

func view(model tea.Model) string {
	return ""
}

func subscriptions(m tea.Model) tea.Subs {
	return nil
}
