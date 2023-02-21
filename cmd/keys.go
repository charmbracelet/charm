package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/charm/client"
	"github.com/charmbracelet/charm/ui/common"
	"github.com/charmbracelet/charm/ui/keys"
	"github.com/spf13/cobra"
)

var (
	randomart    bool
	simpleOutput bool
)

// KeysCmd is the cobra.Command for a user to browser and print their linked
// SSH keys.
var KeysCmd = &cobra.Command{
	Use:   "keys",
	Short: "Browse or print linked SSH keys",
	Long:  paragraph("Charm accounts are powered by " + keyword("SSH keys") + ". This command prints all of the keys linked to your account. To remove keys use the main " + code("charm") + " interface."),
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if common.IsTTY() && !randomart && !simpleOutput {
			// Log to file, if set
			cfg := getCharmConfig()
			if cfg.Logfile != "" {
				f, err := tea.LogToFile(cfg.Logfile, "charm")
				if err != nil {
					return err
				}
				defer f.Close() // nolint:errcheck
			}
			p := keys.NewProgram(cfg)
			if _, err := p.Run(); err != nil {
				return err
			}
			return nil
		}
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
			fp, err := client.FingerprintSHA256(*keys[i])
			if err != nil {
				fp.Value = fmt.Sprintf("Could not generate fingerprint for key %s: %v\n\n", keys[i].Key, err)
			}
			board, err := client.RandomArt(*keys[i])
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
	},
}

func init() {
	KeysCmd.Flags().BoolVarP(&simpleOutput, "simple", "s", false, "simple, non-interactive output (good for scripts)")
	KeysCmd.Flags().BoolVarP(&randomart, "randomart", "r", false, "print SSH 5.1 randomart for each key (the Drunken Bishop algorithm)")
}
