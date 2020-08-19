package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/charm/ui/common"
	"github.com/charmbracelet/charm/ui/link"
	"github.com/charmbracelet/charm/ui/linkgen"
	"github.com/muesli/reflow/indent"
	"github.com/spf13/cobra"
)

var (
	linkCmd = &cobra.Command{
		Use:     "link [code]",
		Short:   "Link multiple machines to your Charm account",
		Long:    formatLong("It’s easy to " + common.Keyword("link") + " multiple machines or keys to your Charm account. Just run " + common.Code("charm link") + " on a machine connected to the account to want to link to start the process."),
		Example: indent.String("charm link\ncharm link XXXXXX", indentBy),
		Args:    cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {

			// Log to file if specified in the environment
			cfg := getCharmConfig()
			if cfg.Logfile != "" {
				f, err := tea.LogToFile(cfg.Logfile, "charm")
				if err != nil {
					return err
				}
				defer f.Close()
			}

			switch len(args) {
			case 0:
				// Initialize a linking session
				p := linkgen.NewProgram(cfg)
				return p.Start()
			default:
				// Join in on a linking session
				p := link.NewProgram(cfg, args[0])
				return p.Start()
			}

		},
	}
)
