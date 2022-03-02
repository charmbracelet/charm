package keys

import (
	"fmt"

	pager "github.com/charmbracelet/bubbles/paginator"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/charm/client"
	charm "github.com/charmbracelet/charm/proto"
	"github.com/charmbracelet/charm/ui/charmclient"
	"github.com/charmbracelet/charm/ui/common"
	"github.com/muesli/reflow/indent"
)

const keysPerPage = 4

type state int

const (
	stateInitCharmClient state = iota
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
func NewProgram(cfg *client.Config) *tea.Program {
	m := NewModel(cfg)
	m.standalone = true
	return tea.NewProgram(m)
}

type (
	keysLoadedMsg  charm.Keys
	unlinkedKeyMsg int
	errMsg         struct {
		err error
	}
)

// Model is the Tea state model for this user interface.
type Model struct {
	cc             *client.Client
	cfg            *client.Config
	styles         common.Styles
	pager          pager.Model
	state          state
	err            error
	standalone     bool
	activeKeyIndex int                // index of the key in the below slice which is currently in use
	keys           []*charm.PublicKey // keys linked to user's account
	index          int                // index of selected key in relation to the current page
	Exit           bool
	Quit           bool
	spinner        spinner.Model
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
	m.pager, _ = m.pager.Update(msg)

	// If selected item is out of bounds, put it in bounds
	numItems := m.pager.ItemsOnPage(len(m.keys))
	m.index = min(m.index, numItems-1)
}

// SetCharmClient sets a pointer to the charm client on the model. The Charm
// Client is necessary for all network-related operations.
func (m *Model) SetCharmClient(cc *client.Client) {
	if cc == nil {
		panic("charm client is nil")
	}
	m.cc = cc
}

// NewModel creates a new model with defaults.
func NewModel(cfg *client.Config) Model {
	st := common.DefaultStyles()

	p := pager.NewModel()
	p.PerPage = keysPerPage
	p.Type = pager.Dots
	p.InactiveDot = st.InactivePagination.Render("•")

	return Model{
		cfg:            cfg,
		styles:         st,
		pager:          p,
		state:          stateLoading,
		err:            nil,
		activeKeyIndex: -1,
		keys:           []*charm.PublicKey{},
		index:          0,
		spinner:        common.NewSpinner(),
		Exit:           false,
		Quit:           false,
	}
}

// Init is the Tea initialization function.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		charmclient.NewClient(m.cfg),
		spinner.Tick,
	)
}

// Update is the tea update function which handles incoming messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
					// The user is going to delete her active key. Double confirm.
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
		m.err = msg.Err
		return m, tea.Quit

	case charmclient.NewClientMsg:
		m.cc = msg
		m.state = stateLoading
		return m, LoadKeys(m)

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
			m.spinner, cmd = m.spinner.Update(msg)
		}
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
func (m Model) View() string {
	if m.err != nil {
		return m.err.Error()
	}

	var s string

	switch m.state {
	case stateInitCharmClient:
		s = m.spinner.View() + " Initializing...\n\n"
	case stateLoading:
		s = m.spinner.View() + " Loading...\n\n"
	case stateQuitting:
		s = "Thanks for using Charm!\n"
	default:
		s = "Here are the keys linked to your Charm account.\n\n"

		// Keys
		s += keysView(m)
		if m.pager.TotalPages > 1 {
			s += m.pager.View()
		}

		// Footer
		switch m.state {
		case stateDeletingKey:
			s += m.promptView("Delete this key?")
		case stateDeletingActiveKey:
			s += m.promptView("This is the key currently in use. Are you, like, for-sure-for-sure?")
		case stateDeletingAccount:
			s += m.promptView("Sure? This will delete your account. Are you absolutely positive?")
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
		s += m.newStyledKey(m.styles, *key, i+start == m.activeKeyIndex).render(state)
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

func (m Model) promptView(prompt string) string {
	st := m.styles.Delete.Copy().MarginTop(2).MarginRight(1)
	return st.Render(prompt) +
		m.styles.DeleteDim.Render("(y/N)")
}

// LoadKeys returns the command necessary for loading the keys.
func LoadKeys(m Model) tea.Cmd {
	if m.standalone {
		return fetchKeys(m.cc)
	}
	return tea.Batch(
		fetchKeys(m.cc),
		spinner.Tick,
	)
}

// fetchKeys loads the current set of keys via the charm client.
func fetchKeys(cc *client.Client) tea.Cmd {
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
