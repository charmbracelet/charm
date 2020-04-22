package username

import (
	"strings"

	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/charm/ui/common"
	"github.com/charmbracelet/tea"
	"github.com/charmbracelet/teaparty/input"
	"github.com/charmbracelet/teaparty/spinner"
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

	inputModel := input.DefaultModel()
	inputModel.CursorColor = fuschia
	inputModel.Placeholder = "divagurl2000"
	inputModel.Prompt = focusedPrompt
	inputModel.CharLimit = 50
	inputModel.Focus()

	spinnerModel := spinner.NewModel()
	spinnerModel.Type = spinner.Dot

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

// UPDATE

func Update(msg tea.Msg, m Model) (Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.Type {

		case tea.KeyCtrlC: // quit
			m.Quit = true
			return m, nil
		case tea.KeyEscape: // exit this mini-app
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
					m.newName = strings.TrimSpace(m.input.Value)
					return m, setName(m) // fire off the command, too
				case cancelButton: // Exit this mini-app
					m.Done = true
					return m, nil
				}
			}

			// Pass messages through to the input element if that's the element
			// in focus
			if m.index == textInput {
				var cmd tea.Cmd
				m.input, cmd = input.Update(msg, m.input)
				return m, cmd
			}

			return m, nil
		}

	case NameTakenMsg:
		m.state = ready
		name := te.String(m.newName).Foreground(color("203")).String()
		m.errMsg = te.String("Sorry,").Foreground(color("241")).String() +
			" " + name + " " +
			te.String("is taken.").Foreground(color("241")).String()
		return m, nil

	case NameInvalidMsg:
		m.state = ready
		m.errMsg = te.String(common.Wrap(
			te.String("Invalid name. ").Foreground(color("203")).String() +
				te.String("Names can only contain plain letters and numbers and must be less than 50 characters. And no emojis, kiddo.").Foreground(color("241")).String(),
		)).Foreground(color("203")).String()
		return m, nil

	case errMsg:
		m.state = ready
		errMsg := common.Wrap(
			te.String("Oh, what? There was a curious error we were not expecting. ").Foreground(color("203")).String() +
				te.String(msg.Error()).Foreground(color("241")).String(),
		)
		m.errMsg = errMsg
		return m, nil

	case spinner.TickMsg:
		m.spinner, _ = spinner.Update(msg, m.spinner)
		return m, nil

	default:
		m.input, _ = input.Update(msg, m.input) // Do we still need this?
		return m, nil
	}
}

// VIEWS

func View(m Model) string {
	s := "Enter a new username\n\n"
	s += input.View(m.input) + "\n\n"
	if m.state == submitting {
		s += spinnerView(m)
	} else {
		s += common.OKButtonView(m.index == 1, true) +
			" " + common.CancelButtonView(m.index == 2, false)
		if m.errMsg != "" {
			s += "\n\n" + m.errMsg
		}
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

func spinnerView(m Model) string {
	return te.String(spinner.View(m.spinner)).Foreground(color("241")).String() +
		" Submitting..."
}

// SUBSCRIPTIONS

// Blink wraps input's Blink subscription
func Blink(model tea.Model) tea.Sub {
	m, ok := model.(Model)
	if !ok {
		return nil // TODO: handle this error properly
	}
	return tea.SubMap(input.Blink, m.input)
}

func Spin(model tea.Model) tea.Sub {
	m, ok := model.(Model)
	if !ok {
		return nil
	}
	if m.state == submitting {
		return tea.SubMap(spinner.Sub, m.spinner)
	}
	return nil
}

// COMMANDS

// Attempt to update the username on the server
func setName(m Model) tea.Cmd {
	return func() tea.Msg {

		// Validate before resetting the session to speed things up and keep us
		// from pounding charm.RenewSession().
		if !charm.ValidateName(m.newName) {
			return NameInvalidMsg{}
		}

		// We must renew the session for every subsequent SSH-backed command we
		// run. In the case below, we request a new JWT when setting the username.
		if err := m.cc.RenewSession(); err != nil {
			return errMsg(err)
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
