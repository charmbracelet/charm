package username

import (
	"strings"

	"github.com/charmbracelet/boba"
	"github.com/charmbracelet/boba/spinner"
	input "github.com/charmbracelet/boba/textinput"
	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/charm/ui/common"
	te "github.com/muesli/termenv"
)

type state int

const (
	ready state = iota
	submitting
)

// index specifies the UI element that's in focus
type index int

const (
	textInput index = iota
	okButton
	cancelButton
)

const (
	prompt = "> "
)

var (
	focusedPrompt = te.String(prompt).Foreground(common.Fuschia.Color()).String()
)

// MSG

// NameSetMsg is sent when a new name has been set successfully. It contains
// the new name.
type NameSetMsg string

// NameTakenMsg is sent when the requested username has already been taken
type NameTakenMsg struct{}

// NameInvalidMsg is sent when the requested username has failed validation
type NameInvalidMsg struct{}

type errMsg error

// MODEL

type Model struct {
	Done bool // true when it's time to exit this view
	Quit bool // true when the user wants to quit the whole program

	cc      *charm.Client
	state   state
	newName string
	index   index
	errMsg  string
	input   input.Model
	spinner spinner.Model
}

// updateFocus updates the focused states in the model based on the current
// focus index.
func (m *Model) updateFocus() {
	if m.index == textInput && !m.input.Focused() {
		m.input.Focus()
		m.input.Prompt = focusedPrompt
	} else if m.index != textInput && m.input.Focused() {
		m.input.Blur()
		m.input.Prompt = prompt
	}
}

// Move the focus index one unit forward
func (m *Model) indexForward() {
	m.index++
	if m.index > cancelButton {
		m.index = textInput
	}
	m.updateFocus()
}

// Move the focus index one unit Backwards
func (m *Model) indexBackward() {
	m.index--
	if m.index < textInput {
		m.index = cancelButton
	}
	m.updateFocus()
}

func NewModel(cc *charm.Client) Model {

	inputModel := input.NewModel()
	inputModel.CursorColor = common.Fuschia.String()
	inputModel.Placeholder = "divagurl2000"
	inputModel.Prompt = focusedPrompt
	inputModel.CharLimit = 50
	inputModel.Focus()

	spinnerModel := spinner.NewModel()
	spinnerModel.Type = spinner.Dot
	spinnerModel.ForegroundColor = common.SpinnerColor

	return Model{
		Done:    false,
		Quit:    false,
		cc:      cc,
		state:   ready,
		newName: "",
		index:   textInput,
		errMsg:  "",
		input:   inputModel,
		spinner: spinnerModel,
	}
}

func Init(cc *charm.Client) func() (boba.Model, boba.Cmd) {
	return func() (boba.Model, boba.Cmd) {
		m := NewModel(cc)
		return m, InitialCmd(m)
	}
}

func InitialCmd(m Model) boba.Cmd {
	return input.Blink(m.input)
}

// UPDATE

func Update(msg boba.Msg, m Model) (Model, boba.Cmd) {
	switch msg := msg.(type) {

	case boba.KeyMsg:
		switch msg.Type {

		case boba.KeyCtrlC: // quit
			m.Quit = true
			return m, nil
		case boba.KeyEscape: // exit this mini-app
			m.Done = true
			return m, nil

		default:
			// Ignore keys if we're submitting
			if m.state == submitting {
				return m, nil
			}

			switch msg.String() {
			case "tab":
				m.indexForward()
			case "shift+tab":
				m.indexBackward()
			case "l":
				fallthrough
			case "k":
				fallthrough
			case "right":
				if m.index != textInput {
					m.indexForward()
				}
			case "h":
				fallthrough
			case "j":
				fallthrough
			case "left":
				if m.index != textInput {
					m.indexBackward()
				}
			case "up":
				fallthrough
			case "down":
				if m.index == textInput {
					m.indexForward()
				} else {
					m.index = textInput
					m.updateFocus()
				}
			case "enter":
				switch m.index {
				case textInput:
					fallthrough
				case okButton: // Submit the form
					m.state = submitting
					m.errMsg = ""
					m.newName = strings.TrimSpace(m.input.Value())
					return m, boba.Batch(
						setName(m), // fire off the command, too
						spinner.Tick(m.spinner),
					)
				case cancelButton: // Exit this mini-app
					m.Done = true
					return m, nil
				}
			}

			// Pass messages through to the input element if that's the element
			// in focus
			if m.index == textInput {
				var cmd boba.Cmd
				m.input, cmd = input.Update(msg, m.input)
				return m, cmd
			}

			return m, nil
		}

	case NameTakenMsg:
		m.state = ready
		m.errMsg = common.Subtle("Sorry, ") +
			te.String(m.newName).Foreground(common.Red.Color()).String() +
			common.Subtle("is taken.")
		return m, nil

	case NameInvalidMsg:
		m.state = ready
		head := te.String("Invalid name. ").Foreground(common.Red.Color()).String()
		body := common.Subtle("Names can only contain plain letters and numbers and must be less than 50 characters. And no emojis, kiddo.")
		m.errMsg = common.Wrap(head + body)
		return m, nil

	case errMsg:
		m.state = ready
		head := te.String("Oh, what? There was a curious error we were not expecting. ").Foreground(common.Red.Color()).String()
		body := common.Subtle(msg.Error())
		m.errMsg = common.Wrap(head + body)
		return m, nil

	case spinner.TickMsg:
		var cmd boba.Cmd
		m.spinner, cmd = spinner.Update(msg, m.spinner)
		return m, cmd

	default:
		var cmd boba.Cmd
		m.input, cmd = input.Update(msg, m.input) // Do we still need this?
		return m, cmd
	}
}

// VIEWS

func View(m Model) string {
	s := "Enter a new username\n\n"
	s += input.View(m.input) + "\n\n"
	if m.state == submitting {
		s += spinnerView(m)
	} else {
		s += common.OKButtonView(m.index == 1, true)
		s += " " + common.CancelButtonView(m.index == 2, false)
		if m.errMsg != "" {
			s += "\n\n" + m.errMsg
		}
	}
	return s
}

func nameSetView(m Model) string {
	return "OK! Your new username is " + m.newName
}

func spinnerView(m Model) string {
	return spinner.View(m.spinner) + " Submitting..."
}

// COMMANDS

// Attempt to update the username on the server
func setName(m Model) boba.Cmd {
	return func() boba.Msg {

		// Validate before resetting the session to potentially save some
		// network traffic and keep things feeling speedy.
		if !charm.ValidateName(m.newName) {
			return NameInvalidMsg{}
		}

		u, err := m.cc.SetName(m.newName)
		if err == charm.ErrNameTaken {
			return NameTakenMsg{}
		} else if err == charm.ErrNameInvalid {
			return NameInvalidMsg{}
		} else if err != nil {
			return errMsg(err)
		}

		return NameSetMsg(u.Name)
	}
}
