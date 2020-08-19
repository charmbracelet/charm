package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/charm/ui/common"
	"github.com/charmbracelet/charm/ui/keys"
	"github.com/spf13/cobra"
)

var (
	keysCmd = &cobra.Command{
		Use:   "keys",
		Short: "Browse or print linked keys",
		Long:  formatLong("Charm accounts are powered by " + common.Keyword("SSH keys") + ". This command prints all of the keys linked to your account. To remove keys use the main " + common.Code("charm") + " interface."),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if isTTY() && !simpleOutput && !randomart {

				// Log to file, if set
				cfg := getCharmConfig()
				if cfg.Logfile != "" {
					f, err := tea.LogToFile(cfg.Logfile, "charm")
					if err != nil {
						return err
					}
					defer f.Close()
				}

				return keys.NewProgram(cfg).Start()

			} else {
				cc := initCharmClient()

				// Print randomart with fingerprints
				k, err := cc.AuthorizedKeysWithMetadata()
				if err != nil {
					return err
				}

				keys := k.Keys
				for i := 0; i < len(keys); i++ {
					if !randomart {
						fmt.Println(keys[i].Key)
						continue
					}
					fp, err := keys[i].FingerprintSHA256()
					if err != nil {
						fp.Value = fmt.Sprintf("Could not generate fingerprint for key %s: %v\n\n", keys[i].Key, err)
					}
					board, err := keys[i].RandomArt()
					if err != nil {
						board = fmt.Sprintf("Could not generate randomart for key %s: %v\n\n", keys[i].Key, err)
					}
					cr := "\n\n"
					if i == len(keys)-1 {
						cr = "\n"
					}
					fmt.Printf("%s\n%s%s", fp, board, cr)
				}
				return nil
			}
		},
	}
)
