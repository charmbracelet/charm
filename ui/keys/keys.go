package keys

import (
	"errors"
	"fmt"

	pager "github.com/charmbracelet/bubbles/paginator"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/charm/ui/charmclient"
	"github.com/charmbracelet/charm/ui/common"
	"github.com/charmbracelet/charm/ui/keygen"
	"github.com/muesli/reflow/indent"
	te "github.com/muesli/termenv"
)

const keysPerPage = 4

type state int

const (
	stateInitCharmClient state = iota
	stateKeygenRunning
	stateKeygenFinished
	stateLoading
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

// NewProgram creates a new Tea program.
func NewProgram(cfg *charm.Config) *tea.Program {
	return tea.NewProgram(Init(cfg), Update, View)
}

type keysLoadedMsg charm.Keys
type unlinkedKeyMsg int
type errMsg struct {
	err error
}

// Model is the Tea state model for this user interface.
type Model struct {
	cc             *charm.Client
	cfg            *charm.Config
	pager          pager.Model
	state          state
	err            error
	standalone     bool
	activeKeyIndex int         // index of the key in the below slice which is currently in use
	keys           []charm.Key // keys linked to user's account
	index          int         // index of selected key in relation to the current page
	Exit           bool
	Quit           bool
	spinner        spinner.Model
	keygen         keygen.Model
}

// getSelectedIndex returns the index of the cursor in relation to the total
// number of items.
func (m *Model) getSelectedIndex() int {
	return m.index + m.pager.Page*m.pager.PerPage
}

// UpdatePaging runs an update against the underlying pagination model as well
// as performing some related tasks on this model.
func (m *Model) UpdatePaging(msg tea.Msg) {
	// Handle paging
	m.pager.SetTotalPages(len(m.keys))
	m.pager, _ = pager.Update(msg, m.pager)

	// If selected item is out of bounds, put it in bounds
	numItems := m.pager.ItemsOnPage(len(m.keys))
	m.index = min(m.index, numItems-1)
}

// SetCharmClient sets a pointer to the charm client on the model. The Charm
// Client is necessary for all network-related operations.
func (m *Model) SetCharmClient(cc *charm.Client) {
	if cc == nil {
		panic("charm client is nil")
	}
	m.cc = cc
}

// NewModel creates a new model with defaults.
func NewModel(cfg *charm.Config) Model {
	p := pager.NewModel()
	p.PerPage = keysPerPage
	p.Type = pager.Dots
	p.InactiveDot = te.String("•").
		Foreground(common.NewColorPair("#4F4F4F", "#CACACA").Color()).
		String()

	s := spinner.NewModel()
	s.Frames = spinner.Dot
	s.ForegroundColor = "241"

	return Model{
		cfg:            cfg,
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

// Init is the Tea initialization function.
func Init(cfg *charm.Config) func() (tea.Model, tea.Cmd) {
	return func() (tea.Model, tea.Cmd) {
		m := NewModel(cfg)
		m.standalone = true
		m.state = stateInitCharmClient
		return m, tea.Batch(
			charmclient.NewClient(cfg),
			spinner.Tick(m.spinner),
		)
	}
}

// Update is the tea update function which handles incoming messages.
func Update(msg tea.Msg, model tea.Model) (tea.Model, tea.Cmd) {
	m, ok := model.(Model)
	if !ok {
		return Model{
			err: errors.New("could not perform assertion on model in keys update"),
		}, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			if m.standalone {
				m.state = stateQuitting
				return m, tea.Quit
			}
			m.Exit = true
			return m, nil

		// Select individual items
		case "up", "k":
			// Move up
			m.index--
			if m.index < 0 && m.pager.Page > 0 {
				m.index = m.pager.PerPage - 1
				m.pager.PrevPage()
			}
			m.index = max(0, m.index)
		case "down", "j":
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
				return m, unlinkKey(m)
			case stateDeletingActiveKey:
				// Active key will be deleted. Remove the key and exit.
				fallthrough
			case stateDeletingAccount:
				// Account will be deleted. Remove the key and exit.
				m.state = stateQuitting
				return m, unlinkKey(m)
			}
		}

	case charmclient.ErrMsg:
		m.err = msg.Err
		return m, tea.Quit

	case charmclient.SSHAuthErrorMsg:
		if m.state == stateInitCharmClient {
			// Couldn't find SSH keys, so let's try the keygen
			m.state = stateKeygenRunning
			m.keygen = keygen.NewModel()
			return m, keygen.GenerateKeys
		}
		// Keygen failed too
		m.err = msg.Err
		return m, tea.Quit

	case charmclient.NewClientMsg:
		m.cc = msg
		m.state = stateLoading
		return m, LoadKeys(m)

	case keygen.DoneMsg:
		m.state = stateKeygenFinished
		return m, charmclient.NewClient(m.cfg)

	case errMsg:
		m.err = msg.err
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
		var cmd tea.Cmd
		if m.state < stateNormal {
			m.spinner, cmd = spinner.Update(msg, m.spinner)
		}
		return m, cmd
	}

	// Update keygen
	if m.state == stateKeygenRunning {
		newKeygenModel, cmd := keygen.Update(msg, m.keygen)
		mdl, ok := newKeygenModel.(keygen.Model)
		if !ok {
			m.err = errors.New("could not perform assertion on keygen model in link update")
			return m, tea.Quit
		}
		m.keygen = mdl
		return m, cmd
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

// View renders the current UI into a string.
func View(model tea.Model) string {
	m, ok := model.(Model)
	if !ok {
		m.err = errors.New("could not perform assertion on model")
	}

	if m.err != nil {
		return m.err.Error()
	}

	var s string

	switch m.state {
	case stateInitCharmClient:
		s = spinner.View(m.spinner) + " Initializing...\n\n"
	case stateKeygenRunning:
		if m.keygen.Status != keygen.StatusSuccess {
			s += spinner.View(m.spinner)
		}
		s += keygen.View(m.keygen)
	case stateLoading:
		s = spinner.View(m.spinner) + " Loading...\n\n"
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
			s += "\n\n" + helpView(m)
		}
	}

	if m.standalone {
		return indent.String(fmt.Sprintf("\n%s\n", s), 2)
	}
	return s
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
	var items []string
	if len(m.keys) > 1 {
		items = append(items, "j/k, ↑/↓: choose")
	}
	if m.pager.TotalPages > 1 {
		items = append(items, "h/l, ←/→: page")
	}
	items = append(items, []string{"x: delete", "esc: exit"}...)
	return common.HelpView(items...)
}

func promptDeleteView() string {
	return te.String("\n\nDelete this key? ").Foreground(common.Red.Color()).String() +
		te.String("(y/N)").Foreground(common.FaintRed.Color()).String()
}

func promptDeleteActiveKeyView() string {
	return te.String("\n\nThis is the key currently in use. Are you, like, for-sure-for-sure? ").Foreground(common.Red.Color()).String() +
		te.String("(y/N)").Foreground(common.FaintRed.Color()).String()
}

func promptDeleteAccountView() string {
	return te.String("\n\nSure? This will delete your account. Are you absolutely positive? ").Foreground(common.Red.Color()).String() +
		te.String("(y/N)").Foreground(common.FaintRed.Color()).String()
}

// LoadKeys returns the command necessary for loading the keys.
func LoadKeys(m Model) tea.Cmd {
	if m.standalone {
		return fetchKeys(m.cc)
	}
	return tea.Batch(
		fetchKeys(m.cc),
		spinner.Tick(m.spinner),
	)
}

// fetchKeys loads the current set of keys via the charm client.
func fetchKeys(cc *charm.Client) tea.Cmd {
	return func() tea.Msg {
		ak, err := cc.AuthorizedKeysWithMetadata()
		if err != nil {
			return errMsg{err}
		}
		return keysLoadedMsg(*ak)
	}
}

// unlinkKey deletes the selected key.
func unlinkKey(m Model) tea.Cmd {
	return func() tea.Msg {
		err := m.cc.UnlinkAuthorizedKey(m.keys[m.getSelectedIndex()].Key)
		if err != nil {
			return errMsg{err}
		}
		return unlinkedKeyMsg(m.index)
	}
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
