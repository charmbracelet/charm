package cmd

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/charm/ui/common"
	"github.com/charmbracelet/charm/ui/keygen"
	"github.com/spf13/cobra"
)

var (
	simpleOutput bool
	randomart    bool

	// KeygenCmd is the cobra.Command to generate a new SSH keypair and user account.
	KeygenCmd = &cobra.Command{
		Use:    "keygen",
		Hidden: true,
		Short:  "Generate SSH keys",
		Long:   paragraph("Charm accounts are powered by " + keyword("SSH keys") + ". This command will create them for you."),
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if common.IsTTY() && !simpleOutput {
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
			}
			return nil
		},
	}
)
