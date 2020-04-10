package keys

import (
	"fmt"
	"time"

	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/charm/ui/common"
	"github.com/charmbracelet/tea"
	"github.com/charmbracelet/teaparty/pager"
	"github.com/muesli/reflow/indent"
	te "github.com/muesli/termenv"
)

const keysPerPage = 4

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

// Model is the Tea state model for this user interface
type Model struct {
	cc           *charm.Client
	pager        pager.Model
	standalone   bool
	keys         []charm.Key
	index        int
	promptDelete bool // have we prompted to delete the item at the current index?
	Exit         bool
	Quit         bool
}

func (m *Model) UpdatePaging(msg tea.Msg) {

	// Handle paging
	m.pager.SetTotalPages(len(m.keys))
	m.pager, _ = pager.Update(msg, m.pager)

	// If selected item is out of bounds, put it in bounds
	numItems := m.pager.ItemsOnPage(len(m.keys))
	m.index = min(m.index, numItems-1)
}

// Init is the Tea initialization function which returns an initial model and,
// potentially, an initial command
func Init(cc *charm.Client) func() (tea.Model, tea.Cmd) {
	return func() (tea.Model, tea.Cmd) {
		m := NewModel(cc)
		m.standalone = true
		return m, nil
	}
}

// NewModel creates a new model with defaults
func NewModel(cc *charm.Client) Model {
	p := pager.NewModel()
	p.PerPage = keysPerPage
	p.InactiveDot = common.Subtle("•")
	p.Type = pager.Dots

	now := time.Now()
	keys := []charm.Key{
		charm.Key{"AAAAB3NzaC1yc2EAAAADAQABAAABgQDQ3mp1RT5MKT4i6ROZ59NQDds5IJR8c7Lm0Vc5p+E0LFTPveP/crWWpfdZNvYSu0nmAuRVcbtyol9y5Q7/WVoGdu5zi4f+mRzsUMFsgeKBqJNeJ+X8Y8kpcP6k78iYN82sROk/WddtnwXfEw+rsaAyYe7h4Hp4QZ/GCYCpi5KzNfbg0/rOlllPu387izcftixtXelO4JccaGfycgeQH8ylWpB5cqrK9Uqj86PykkwsTEM4MGJGU4FYnhl5RMxzPkd9Xt09YBZJHErbFjX9peSVCPQe7GUi+YDtxR1C4XRfNeXQq8AYiPbW82Y1Q00EInvDPAVv2+CQs/pKSAmWX61G5T2LbuXpNuwD2M7Jps1UlLCgPKE6G4J6I5TT5LOkwQNDiZxv6LGMhco7ji/jkFpXIvc0yFEH8zzswgheDEqnv9nqMQ6C6XcP0Au6ygThspn/eC75ZxLpd624Dvne9JRtpmPdVb45+k1dyDMYwf3vCgULXYrPVU7JZctu8dkzjQE=", &now},
		charm.Key{"hej", &now},
		charm.Key{"hallo", &now},
		charm.Key{"konnichiwa", &now},
		charm.Key{"annyeong", &now},
		charm.Key{"hola", &now},
	}
	p.SetTotalPages(len(keys))

	return Model{
		cc:    cc,
		pager: p,
		keys:  keys,
		index: 0,
		Exit:  false,
		Quit:  false,
	}
}

// Update is the Tea update function which handles incoming IO
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
			m.Quit = true
			if m.standalone {
				return m, tea.Quit
			}
			return m, nil
		case "q":
			fallthrough
		case "esc":
			if m.standalone {
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
			numItems := m.pager.ItemsOnPage(len(m.keys))
			m.index++
			if m.index > numItems-1 && m.pager.Page < m.pager.TotalPages-1 {
				m.index = 0
				m.pager.NextPage()
			}
			m.index = min(numItems-1, m.index)

		// Delete
		case "x":
			m.promptDelete = true
			m.UpdatePaging(msg)
			return m, nil

			// Confirm Delete
		case "y":
			if m.promptDelete {
				// TODO: return deletion command, actually delete, and so on
				m.promptDelete = false
				return m, nil
			}
		}
	}

	m.UpdatePaging(msg)

	// If an item is being confirmed for delete, any key (other than the key
	// used for confirmation above) cancels the deletion
	k, ok := msg.(tea.KeyMsg)
	if ok && k.String() != "x" {
		m.promptDelete = false
	}

	return m, nil
}

// View renders the current UI into a string
func View(model tea.Model) string {
	m, ok := model.(Model)
	if !ok {
		// TODO: handle error
		return ""
	}
	s := "Here are the keys linked to your Charm account.\n\n"
	s += keysView(m)
	if m.pager.TotalPages > 1 {
		s += pager.View(m.pager)
	}
	if m.promptDelete {
		s += promptDeleteView()
	} else {
		s += helpView()
	}
	s = fmt.Sprintf("%s\n", s)
	if m.standalone {
		return indent.String("\n"+s, 2)
	}
	return s
}

func keysView(m Model) string {
	if len(m.keys) == 0 {
		return ""
	}

	var (
		s          string
		state      = keyNormal
		start, end = m.pager.GetSliceBounds(len(m.keys))
		slice      = m.keys[start:end]
	)

	// Render key info
	for i, key := range slice {
		if m.promptDelete && m.index == i {
			state = keyDeleting
		} else if m.index == i {
			state = keySelected
		} else {
			state = keyNormal
		}
		s += newStyledKey(key).render(state)
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

func helpView() string {
	return common.HelpView("j/k, ↑/↓: choose • h/l, ←/→: page, x: delete, esc: exit")
}

func promptDeleteView() string {
	return te.String("\n\nDelete this key? ").Foreground(hotPink).String() +
		te.String("(y/N)").Foreground(dullHotPink).String()
}

func Subscriptions(model tea.Model) tea.Subs {
	return nil
}

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
