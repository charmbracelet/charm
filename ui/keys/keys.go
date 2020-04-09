package keys

import (
	"fmt"
	"time"

	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/charm/ui/common"
	"github.com/charmbracelet/tea"
	"github.com/charmbracelet/teaparty/pager"
	"github.com/muesli/reflow/indent"
)

const keysPerPage = 4

// NewProgram creates a new Tea program
func NewProgram(cc *charm.Client) *tea.Program {
	return tea.NewProgram(Init(cc), Update, View, nil)
}

// Model is the Tea state model for this user interface
type Model struct {
	cc         *charm.Client
	pager      pager.Model
	standalone bool
	keys       []charm.Key
}

// Init is the Tea initialization function which returns an initial model and,
// potentially, an initial command
func Init(cc *charm.Client) func() (tea.Model, tea.Cmd) {
	return func() (tea.Model, tea.Cmd) {
		now := time.Now()

		m := NewModel(cc)
		m.standalone = true
		m.keys = []charm.Key{
			charm.Key{"hey", &now},
			charm.Key{"yo", &now},
			charm.Key{"hallo", &now},
			charm.Key{"konnichiwa", &now},
			charm.Key{"annyeong", &now},
			charm.Key{"hola", &now},
		}
		return m, nil
	}
}

// NewModel creates a new model with defaults
func NewModel(cc *charm.Client) Model {
	p := pager.NewModel()
	p.PerPage = keysPerPage
	p.InactiveDot = common.Subtle("•")
	p.Type = pager.Dots
	return Model{
		cc:    cc,
		pager: p,
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
			fallthrough
		case "q":
			fallthrough
		case "esc":
			return m, tea.Quit
		}
	}

	m.pager.SetTotalPages(len(m.keys))
	m.pager, _ = pager.Update(msg, m.pager)

	return m, nil
}

// View renders the current UI into a string
func View(model tea.Model) string {
	m, ok := model.(Model)
	if !ok {
		// TODO: handle error
		return ""
	}
	s := keysView(m)
	s += pager.View(m.pager)
	return "\n" + indent.String(s+helpView(), 2)
}

func keysView(m Model) string {
	if len(m.keys) == 0 {
		return ""
	}
	var (
		s          string
		start, end = m.pager.GetSliceBounds(len(m.keys))
		slice      = m.keys[start:end]
	)
	for _, v := range slice {
		s += fmt.Sprintf("%s\n\n", keyView(v))
	}
	if len(slice) < keysPerPage {
		for i := len(slice); i < keysPerPage; i++ {
			s += "\n\n\n"
		}
	}
	return s
}

func keyView(key charm.Key) string {
	return common.KeyValueView("Key", key.Key, "Created", key.CreatedAt.Format("Mon 2 Jan 2006 15:04:05 MST"))
}

func helpView() string {
	return common.HelpView("j/k, ↑/↓: choose • h/l, ←/→: page, x: delete, esc: exit")
}
