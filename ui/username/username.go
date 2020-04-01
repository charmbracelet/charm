package username

import (
	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/tea"
	"github.com/charmbracelet/teaparty/input"
	"github.com/muesli/reflow/wordwrap"
	te "github.com/muesli/termenv"
)

const (
	prompt = "> "
)

var (
	color         = te.ColorProfile().Color
	fuschia       = "#EE6FF8"
	yellowGreen   = "#ECFD65"
	focusedPrompt = te.String(prompt).Foreground(color(fuschia)).String()
)

type index int

const (
	textInput index = iota
	okButton
	cancelButton
)

// MSG

type NameSetMsg struct{}

type NameTakenMsg struct{}

type NameInvalidMsg struct{}

type ExitMsg struct{}

// MODEL

type Model struct {
	cc      *charm.Client
	newName string
	input   input.Model
	index   index
	errMsg  string
}

// Reset the model to its default state
func (m *Model) reset() Model {
	return NewModel(m.cc)
}

// INIT

func NewModel(cc *charm.Client) Model {
	inputModel := input.DefaultModel()
	inputModel.CursorColor = fuschia
	inputModel.Placeholder = "divagurl2000"
	inputModel.Focus()
	inputModel.Prompt = focusedPrompt

	return Model{
		cc:      cc,
		newName: "",
		input:   inputModel,
		index:   textInput,
		errMsg:  "",
	}
}

// UPDATE

func Update(msg tea.Msg, m Model) (Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch key := msg.Type; key {

		case tea.KeyTab:
			fallthrough
		case tea.KeyShiftTab:

			// Set focus index
			if key == tea.KeyTab {
				m.index++
				if m.index > cancelButton {
					m.index = textInput
				}
			} else {
				m.index--
				if m.index < textInput {
					m.index = cancelButton
				}
			}

			// Set focus/blur on input field
			if m.index == textInput && !m.input.Focused() {
				m.input.Focus()
				m.input.Prompt = focusedPrompt
			} else if m.index != textInput && m.input.Focused() {
				m.input.Blur()
				m.input.Prompt = prompt
			}

			return m, nil

		case tea.KeyEnter:
			switch m.index {
			case textInput:
				m.errMsg = ""
				m.newName = m.input.Value
				return m, tea.CmdMap(setName, m) // fire off the command, too
			case okButton:
				m.errMsg = ""
				m.newName = m.input.Value
				return m, tea.CmdMap(setName, m)
			default: // cancel/exit
				return m.reset(), exit
			}

		case tea.KeyEscape:
			return m.reset(), exit

		default:
			if m.index == textInput {
				var cmd tea.Cmd
				m.input, cmd = input.Update(msg, m.input)
				return m, cmd
			}
			return m, nil
		}

	case NameTakenMsg:
		name := te.String(m.newName).Foreground(color(fuschia)).String()
		m.errMsg = te.String("Sorry,").Foreground(color("203")).String() +
			name +
			te.String(" is taken.").Foreground(color("203")).String()
		return m, nil

	case NameInvalidMsg:
		m.errMsg = te.String(wordwrap.String(
			"Oh. That's an invalid username. Usernames can only contain plain letters and numbers and must be less than 64 characters. And no emojis, kid!",
			50,
		)).Foreground(color("203")).String()
		return m, nil

	case tea.ErrMsg:
		errMsg := wordwrap.String(
			"Oh, what? There was a curious error we were not expecting. ["+msg.String()+"]",
			50,
		)
		m.errMsg = te.String(errMsg).Foreground(color("203")).String()
		return m, nil

	case NameSetMsg:
		// TODO: something. anything.
		return m, nil

	default:
		m.input, _ = input.Update(msg, m.input)
		return m, nil
	}
}

// VIEWS

func View(m Model) string {
	s := "Enter a new username\n\n"
	s += input.View(m.input) + "\n\n"
	s += buttonView("OK", m.index == 1, true) + " " + buttonView("Cancel", m.index == 2, false)
	if m.errMsg != "" {
		s += "\n\n" + m.errMsg
	}
	return s
}

func buttonView(label string, active bool, signalDefault bool) string {
	c := "238"
	if active {
		c = fuschia
	}
	text := te.String(label).Background(color(c))
	if signalDefault {
		text = text.Underline()
	}
	padding := te.String("  ").Background(color(c)).String()
	return padding + text.String() + padding
}

func nameSetView(m Model) string {
	return "OK! Your new username is " + m.newName
}

// SUBSCRIPTIONS

// Blink wraps input's Blink subscription
func Blink(model tea.Model) tea.Sub {
	m, ok := model.(Model)
	if !ok {
		// TODO: handle this error properly
		return nil
	}
	return func(_ tea.Model) tea.Msg {
		return input.Blink(m.input)
	}
}

// COMMANDS

// Attempt to update the username on the server
func setName(model tea.Model) tea.Msg {
	m, ok := model.(Model)
	if !ok {
		return tea.ModelAssertionErr
	}

	_, err := m.cc.SetName(m.newName)
	if err == charm.ErrNameTaken {
		return NameTakenMsg{}
	} else if err == charm.ErrNameInvalid {
		return NameInvalidMsg{}
	} else if err != nil {
		return tea.NewErrMsgFromErr(err)
	}

	return NameSetMsg{}
}

// A command to exit this view
func exit(_ tea.Model) tea.Msg {
	return ExitMsg{}
}
