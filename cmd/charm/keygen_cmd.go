package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/charm/client/ui/common"
	"github.com/charmbracelet/charm/client/ui/keygen"
	"github.com/spf13/cobra"
)

var (
	simpleOutput bool
	randomart    bool

	keygenCmd = &cobra.Command{
		Use:    "keygen",
		Hidden: true,
		Short:  "Generate SSH keys",
		Long:   formatLong("Charm accounts are powered by " + common.Keyword("SSH keys") + ". This command will create them for you."),
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if isTTY() && !simpleOutput {

				// Log to file if specified in the environment
				cfg := getCharmConfig()
				if cfg.Logfile != "" {
					f, err := tea.LogToFile(cfg.Logfile, "charm")
					if err != nil {
						return err
					}
					defer f.Close()
				}

				return keygen.NewProgram(true).Start()
			} else {
				// TODO
			}

			return nil
		},
	}
)

func init() {
	keysCmd.Flags().BoolVarP(&simpleOutput, "simple", "s", false, "simple, non-interactive output (good for scripts)")
	keysCmd.Flags().BoolVarP(&randomart, "randomart", "r", false, "print SSH 5.1 randomart for each key (the Drunken Bishop algorithm)")
}
