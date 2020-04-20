package keys

import (
	"fmt"

	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/charm/ui/common"
	"github.com/charmbracelet/tea"
	"github.com/charmbracelet/teaparty/pager"
	"github.com/charmbracelet/teaparty/spinner"
	"github.com/muesli/reflow/indent"
	te "github.com/muesli/termenv"
)

const keysPerPage = 4

type state int

const (
	stateLoading state = iota
	stateNormal
	stateDeletingKey
	stateDeletingActiveKey
	stateDeletingAccount
	stateQuitting
)

type keyState int

const (
	keyNormal keyState = iota
	keySelected
	keyDeleting
)

// NewProgram creates a new Tea program
func NewProgram(cc *charm.Client) *tea.Program {
	return tea.NewProgram(Init(cc), Update, View, Subscriptions)
}

// MSG

type keysLoadedMsg charm.Keys
type unlinkedKeyMsg int

// MODEL

// Model is the Tea state model for this user interface
type Model struct {
	cc             *charm.Client
	pager          pager.Model
	state          state
	err            error
	standalone     bool
	activeKeyIndex int         // index of the key in the below slice which is currently in use
	keys           []charm.Key // keys linked to user's account
	index          int         // index of selected key in relation to the current page
	spinner        spinner.Model
	Exit           bool
	Quit           bool
}

// getSelectedIndex returns the index of the cursor in relation to the total
// number of items.
func (m *Model) getSelectedIndex() int {
	return m.index + m.pager.Page*m.pager.PerPage
}

func (m *Model) UpdatePaging(msg tea.Msg) {

	// Handle paging
	m.pager.SetTotalPages(len(m.keys))
	m.pager, _ = pager.Update(msg, m.pager)

	// If selected item is out of bounds, put it in bounds
	numItems := m.pager.ItemsOnPage(len(m.keys))
	m.index = min(m.index, numItems-1)
}

// NewModel creates a new model with defaults
func NewModel(cc *charm.Client) Model {
	p := pager.NewModel()
	p.PerPage = keysPerPage
	p.InactiveDot = common.Subtle("•")
	p.Type = pager.Dots

	s := spinner.NewModel()
	s.Type = spinner.Dot
	s.ForegroundColor = "241"

	return Model{
		cc:             cc,
		pager:          p,
		state:          stateLoading,
		err:            nil,
		activeKeyIndex: -1,
		keys:           []charm.Key{},
		index:          0,
		spinner:        s,
		Exit:           false,
		Quit:           false,
	}
}

// INIT

// Init is the Tea initialization function which returns an initial model and,
// potentially, an initial command
func Init(cc *charm.Client) func() (tea.Model, tea.Cmd) {
	return func() (tea.Model, tea.Cmd) {
		m := NewModel(cc)
		m.standalone = true
		return m, LoadKeys
	}
}

// UPDATE

// Update is the Tea update function which handles incoming messages
func Update(msg tea.Msg, model tea.Model) (tea.Model, tea.Cmd) {
	m, ok := model.(Model)
	if !ok {
		// TODO: handle error
		return model, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {

		case "ctrl+c":
			fallthrough
		case "q":
			fallthrough
		case "esc":
			if m.standalone {
				m.state = stateQuitting
				return m, tea.Quit
			}
			m.Exit = true
			return m, nil

		// Select individual items
		case "up":
			fallthrough
		case "k":
			// Move up
			m.index--
			if m.index < 0 && m.pager.Page > 0 {
				m.index = m.pager.PerPage - 1
				m.pager.PrevPage()
			}
			m.index = max(0, m.index)
		case "down":
			fallthrough
		case "j":
			// Move down
			itemsOnPage := m.pager.ItemsOnPage(len(m.keys))
			m.index++
			if m.index > itemsOnPage-1 && m.pager.Page < m.pager.TotalPages-1 {
				m.index = 0
				m.pager.NextPage()
			}
			m.index = min(itemsOnPage-1, m.index)

		// Delete
		case "x":
			m.state = stateDeletingKey
			m.UpdatePaging(msg)
			return m, nil

			// Confirm Delete
		case "y":
			switch m.state {
			case stateDeletingKey:
				if len(m.keys) == 1 {
					// The user is about to delete her account. Double confirm.
					m.state = stateDeletingAccount
					return m, nil
				}
				if m.getSelectedIndex() == m.activeKeyIndex {
					// The user is going to delete
					m.state = stateDeletingActiveKey
					return m, nil
				}
				m.state = stateNormal
				return m, tea.CmdMap(unlinkKey, m)
			case stateDeletingActiveKey:
				// Active key will be deleted. Remove the key and exit.
				fallthrough
			case stateDeletingAccount:
				// Account will be deleted. Remove the key and exit.
				m.state = stateQuitting
				return m, tea.CmdMap(unlinkKey, m)
			}
		}

	case tea.ErrMsg:
		m.err = msg
		return m, nil

	case keysLoadedMsg:
		m.state = stateNormal
		m.index = 0
		m.activeKeyIndex = msg.ActiveKey
		m.keys = msg.Keys

	case unlinkedKeyMsg:
		if m.state == stateQuitting {
			return m, tea.Quit
		}
		i := m.getSelectedIndex()

		// Remove key from array
		m.keys = append(m.keys[:i], m.keys[i+1:]...)

		// Update pagination
		m.pager.SetTotalPages(len(m.keys))
		m.pager.Page = min(m.pager.Page, m.pager.TotalPages-1)

		// Update cursor
		m.index = min(m.index, m.pager.ItemsOnPage(len(m.keys)-1))

		return m, nil

	case spinner.TickMsg:
		m.spinner, _ = spinner.Update(msg, m.spinner)
		return m, nil
	}

	m.UpdatePaging(msg)

	// If an item is being confirmed for delete, any key (other than the key
	// used for confirmation above) cancels the deletion
	k, ok := msg.(tea.KeyMsg)
	if ok && k.String() != "x" {
		m.state = stateNormal
	}

	return m, nil
}

// VIEW

// View renders the current UI into a string
func View(model tea.Model) string {
	m, ok := model.(Model)
	if !ok {
		// TODO: handle error
		return ""
	}

	if m.err != nil {
		return m.err.Error()
	}

	var s string

	switch m.state {
	case stateLoading:
		s = loadingView(m)
	case stateQuitting:
		s = "Thanks for using Charm!\n"
	default:
		s = "Here are the keys linked to your Charm account.\n\n"

		// Keys
		s += keysView(m)
		if m.pager.TotalPages > 1 {
			s += pager.View(m.pager)
		}

		// Footer
		switch m.state {
		case stateDeletingKey:
			s += promptDeleteView()
		case stateDeletingActiveKey:
			s += promptDeleteActiveKeyView()
		case stateDeletingAccount:
			s += promptDeleteAccountView()
		default:
			s += helpView(m)
		}

	}

	if m.standalone {
		return indent.String(fmt.Sprintf("\n%s\n", s), 2)
	}
	return s
}

func loadingView(m Model) string {
	return fmt.Sprintf("%s Loading...\n\n", spinner.View(m.spinner))
}

func keysView(m Model) string {
	var (
		s          string
		state      keyState
		start, end = m.pager.GetSliceBounds(len(m.keys))
		slice      = m.keys[start:end]
	)

	destructiveState :=
		(m.state == stateDeletingKey ||
			m.state == stateDeletingActiveKey ||
			m.state == stateDeletingAccount)

	// Render key info
	for i, key := range slice {
		if destructiveState && m.index == i {
			state = keyDeleting
		} else if m.index == i {
			state = keySelected
		} else {
			state = keyNormal
		}
		s += newStyledKey(key, i+start == m.activeKeyIndex).render(state)
	}

	// If there aren't enough keys to fill the view, fill the missing parts
	// with whitespace
	if len(slice) < m.pager.PerPage {
		for i := len(slice); i < keysPerPage; i++ {
			s += "\n\n\n"
		}
	}

	return s
}

func helpView(m Model) string {
	var s string
	if len(m.keys) > 1 {
		s += "j/k, ↑/↓: choose • "
	}
	if m.pager.TotalPages > 1 {
		s += "h/l, ←/→: page • "
	}
	s += "x: delete • "
	return common.HelpView(s + "esc: exit")
}

func promptDeleteView() string {
	return te.String("\n\nDelete this key? ").Foreground(hotPink).String() +
		te.String("(y/N)").Foreground(dullHotPink).String()
}

func promptDeleteActiveKeyView() string {
	return te.String("\n\nThis is the key currently in use. Are you, like, for-sure-for-sure? ").Foreground(hotPink).String() +
		te.String("(y/N)").Foreground(dullHotPink).String()
}

func promptDeleteAccountView() string {
	return te.String("\n\nSure? This will delete your account. Are you absolutely positive? ").Foreground(hotPink).String() +
		te.String("(y/N)").Foreground(dullHotPink).String()
}

// SUBSCRIPTIONS

func Subscriptions(model tea.Model) tea.Subs {
	m, ok := model.(Model)
	if !ok {
		return nil
	}
	if m.state == stateLoading {
		return tea.Subs{
			"spinner-tick": Spin(m),
		}
	}
	return nil
}

func Spin(model tea.Model) tea.Sub {
	m, ok := model.(Model)
	if !ok {
		return nil
	}
	if m.state == stateLoading {
		return tea.SubMap(spinner.Sub, m.spinner)
	}
	return nil
}

// COMMANDS

// LoadKeys loads the current set of keys from the server
func LoadKeys(model tea.Model) tea.Msg {
	m, ok := model.(Model)
	if !ok {
		return tea.ModelAssertionErr
	}
	m.cc.RenewSession()
	ak, err := m.cc.AuthorizedKeysWithMetadata()
	if err != nil {
		return tea.NewErrMsgFromErr(err)
	}
	return keysLoadedMsg(*ak)
}

// unlinkKey deletes the selected key
func unlinkKey(model tea.Model) tea.Msg {
	m, ok := model.(Model)
	if !ok {
		return tea.ModelAssertionErr
	}
	m.cc.RenewSession()
	err := m.cc.UnlinkAuthorizedKey(m.keys[m.getSelectedIndex()].Key)
	if err != nil {
		return tea.NewErrMsgFromErr(err)
	}
	return unlinkedKeyMsg(m.index)
}

// Utils

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
