package cmd

import (
	"fmt"

	"github.com/charmbracelet/charm/ui/common"
	"github.com/spf13/cobra"
)

// KeySyncCmd is the cobra.Command to rencrypt and sync all encrypt keys for a
// user.
var KeySyncCmd = &cobra.Command{
	Use:    "sync-keys",
	Hidden: true,
	Short:  "Re-encrypt encrypt keys for all linked public keys",
	Long:   common.FormatLong(fmt.Sprintf("%s encrypt keys for all linked public keys", common.Keyword("Re-encrypt"))),
	Args:   cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cc := initCharmClient(animatedKeygen)
		return cc.SyncEncryptKeys()
	},
}
