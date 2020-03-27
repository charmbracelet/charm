package main

import (
	"github.com/charmbracelet/tea"
	"github.com/charmbracelet/teaparty/input"
)

type Model struct {
	usernameInput input.Model
}

func initialize() (tea.Model, tea.Cmd) {
	usernameInput := input.DefaultModel()
	usernameInput.Placeholder = "Fran"
	usernameInput.Focus()

	m := Model{
		usernameInput: usernameInput,
	}
	return m, nil
}

func update(msg tea.Msg, model tea.Model) (tea.Model, tea.Cmd) {

	// TODO: handle this
	m, _ := model.(Model)

	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		default:
			var cmd tea.Cmd
			m.usernameInput, cmd = input.Update(msg, m.usernameInput)
			return m, cmd
		}

	default:
		var cmd tea.Cmd
		m.usernameInput, cmd = input.Update(msg, m.usernameInput)
		return m, cmd

	}
}

func view(model tea.Model) string {
	m, ok := model.(Model)
	if !ok {
		return tea.ModelAssertionErr.Error()
	}

	return input.View(m.usernameInput)
}

func subscriptions(model tea.Model) tea.Subs {
	return tea.Subs{
		"blink": func(mdl tea.Model) tea.Msg {
			m, ok := mdl.(Model)
			if !ok {
				return tea.ModelAssertionErr
			}
			return input.Blink(m.usernameInput)
		},
	}
}
