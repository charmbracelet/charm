package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/charm/ui/common"
	"github.com/charmbracelet/charm/ui/keygen"
	"github.com/spf13/cobra"
)

var (
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

				return tea.NewProgram(keygen.Init, keygen.Update, keygen.View).Start()
			} else {
				// TODO
			}

			return nil
		},
	}
)
