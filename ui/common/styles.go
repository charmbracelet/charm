package common

import (
	"github.com/charmbracelet/lipgloss"
)

// Color definitions.
var (
	indigo       = lipgloss.AdaptiveColor{Light: "#5A56E0", Dark: "#7571F9"}
	subtleIndigo = lipgloss.AdaptiveColor{Light: "#7D79F6", Dark: "#514DC1"}
	cream        = lipgloss.AdaptiveColor{Light: "#FFFDF5", Dark: "#FFFDF5"}
	fuschia      = lipgloss.AdaptiveColor{Light: "#EE6FF8", Dark: "#EE6FF8"}
	green        = lipgloss.Color("#04B575")
	red          = lipgloss.AdaptiveColor{Light: "#FF4672", Dark: "#ED567A"}
	faintRed     = lipgloss.AdaptiveColor{Light: "#FF6F91", Dark: "#C74665"}
)

// Styles describes style definitions for various portions of the Charm TUI.
type Styles struct {
	Cursor,
	Wrap,
	Paragraph,
	Keyword,
	Code,
	Subtle,
	Error,
	Prompt,
	FocusedPrompt,
	Note,
	NoteDim,
	Delete,
	DeleteDim,
	Label,
	LabelDim,
	ListKey,
	ListDim,
	InactivePagination,
	SelectionMarker,
	SelectedMenuItem,
	Checkmark,
	Logo,
	App lipgloss.Style
}

// DefaultStyles returns default styles for the Charm TUI.
func DefaultStyles() Styles {
	s := Styles{}

	s.Cursor = lipgloss.NewStyle().Foreground(fuschia)
	s.Wrap = lipgloss.NewStyle().Width(58)
	s.Keyword = lipgloss.NewStyle().Foreground(green)
	s.Paragraph = s.Wrap.Copy().Margin(1, 0, 0, 2)
	s.Code = lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#FF4672", Dark: "#ED567A"}).
		Background(lipgloss.AdaptiveColor{Light: "#EBE5EC", Dark: "#2B2A2A"}).
		Padding(0, 1)
	s.Subtle = lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#9B9B9B", Dark: "#5C5C5C"})
	s.Error = lipgloss.NewStyle().Foreground(red)
	s.Prompt = lipgloss.NewStyle().MarginRight(1).SetString(">")
	s.FocusedPrompt = s.Prompt.Copy().Foreground(fuschia)
	s.Note = lipgloss.NewStyle().Foreground(green)
	s.NoteDim = lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#ABE5D1", Dark: "#2B4A3F"})
	s.Delete = s.Error.Copy()
	s.DeleteDim = lipgloss.NewStyle().Foreground(faintRed)
	s.Label = lipgloss.NewStyle().Foreground(fuschia)
	s.LabelDim = lipgloss.NewStyle().Foreground(indigo)
	s.ListKey = lipgloss.NewStyle().Foreground(indigo)
	s.ListDim = lipgloss.NewStyle().Foreground(subtleIndigo)
	s.InactivePagination = lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#CACACA", Dark: "#4F4F4F"})
	s.SelectionMarker = lipgloss.NewStyle().
		Foreground(fuschia).
		PaddingRight(1).
		SetString(">")
	s.Checkmark = lipgloss.NewStyle().
		SetString("✔").
		Foreground(green)
	s.SelectedMenuItem = lipgloss.NewStyle().Foreground(fuschia)
	s.Logo = lipgloss.NewStyle().
		Foreground(cream).
		Background(lipgloss.Color("#5A56E0")).
		Padding(0, 1).
		SetString("Charm")
	s.App = lipgloss.NewStyle().Margin(1, 0, 1, 2)

	return s
}
