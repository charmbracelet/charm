package username

import (
	"errors"

	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/tea"
	"github.com/charmbracelet/teaparty/input"
	"github.com/muesli/termenv"
)

var (
	color = termenv.ColorProfile().Color
)

type state int

const (
	nameNotChosen state = iota
	nameTaken
	nameInvalid
	nameSet
	unknownError
)

type index int

const (
	textInput index = iota
	okButton
	cancelButton
)

// MSG

type NameSetMsg struct{}

type ErrorMsg error

type ExitMsg struct{}

// MODEL

type Model struct {
	cc      *charm.Client
	state   state
	newName string
	input   input.Model
	index   index
	err     error
}

// INIT

func NewModel(cc *charm.Client) Model {
	inputModel := input.DefaultModel()

	return Model{
		cc:      cc,
		state:   nameNotChosen,
		newName: "",
		input:   inputModel,
		index:   textInput,
		err:     nil,
	}
}

// UPDATE

func Update(msg tea.Msg, m Model) (Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.Type {

		case tea.KeyTab:
			m.index++
			if m.index > cancelButton {
				m.index = textInput
			}
			if m.index == textInput {
				m.input.Focus()
			}
			return m, nil

		case tea.KeyShiftTab:
			m.index--
			if m.index < textInput {
				m.index = cancelButton
			}
			if m.index != textInput && m.input.Focused() {
				m.input.Blur()
			}
			return m, nil

		default:
			if m.index == textInput {
				var cmd tea.Cmd
				m.input, cmd = input.Update(msg, m.input)
				return m, cmd
			}
			return m, nil
		}

	case ErrorMsg:
		switch msg {
		case charm.ErrNameTaken:
			m.state = nameTaken
			return m, nil
		default:
			m.state = unknownError
			err, ok := msg.(error)
			if !ok {
				m.err = errors.New("very, very unknown error")
			}
			m.err = err
			return m, nil
		}

	case NameSetMsg:
		m.state = nameSet
		return m, nil

	default:
		return m, nil
	}
}

// VIEWS

func View(m Model) string {
	switch m.state {
	case nameNotChosen:
		return setNameView(m)
	default:
		return ""
	}
}

func setNameView(m Model) string {
	s := "Enter a new username:\n\n"
	s += input.View(m.input) + "\n"
	s += buttonView("OK", m.index == 1) + " " + buttonView("Cancel", m.index == 2)
	return s
}

func buttonView(label string, active bool) string {
	s := "  " + label + "  "
	c := "241"
	if active {
		c = "200"
	}
	return termenv.String(s).Background(color(c)).String()
}

func nameSetView(m Model) string {
	return "OK! Your new username is " + m.newName
}

// COMMANDS

func setName(model tea.Model) tea.Msg {
	m, ok := model.(Model)
	if !ok {
		return tea.ModelAssertionErr
	}

	_, err := m.cc.SetName(m.newName)
	if err != nil {
		return ErrorMsg(err)
	}
	return NameSetMsg{}
}
