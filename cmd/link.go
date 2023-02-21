package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/charm/ui/link"
	"github.com/charmbracelet/charm/ui/linkgen"
	"github.com/muesli/reflow/indent"
	"github.com/spf13/cobra"
)

// LinkCmd is the cobra.Command to manage user account linking. Pass the name
// of the parent command.
func LinkCmd(parentName string) *cobra.Command {
	return &cobra.Command{
		Use:     "link [code]",
		Short:   "Link multiple machines to your Charm account",
		Long:    paragraph("It’s easy to " + keyword("link") + " multiple machines or keys to your Charm account. Just run " + code(parentName+" link") + " on a machine connected to the account to want to link to start the process."),
		Example: indent.String(fmt.Sprintf("%s link\b%s link XXXXXX", parentName, parentName), 2),
		Args:    cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Log to file if specified in the environment
			cfg := getCharmConfig()
			if cfg.Logfile != "" {
				f, err := tea.LogToFile(cfg.Logfile, "charm")
				if err != nil {
					return err
				}
				defer f.Close() //nolint:errcheck
			}

			var p *tea.Program
			switch len(args) {
			case 0:
				// Initialize a linking session
				p = linkgen.NewProgram(cfg, parentName)
			default:
				// Join in on a linking session
				p = link.NewProgram(cfg, args[0])
			}
			if _, err := p.Run(); err != nil {
				return err
			}
			return nil
		},
	}
}
