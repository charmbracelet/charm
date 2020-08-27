package main

import (
	"fmt"

	"github.com/charmbracelet/charm/ui/common"
	"github.com/spf13/cobra"
)

var (
	keySyncCmd = &cobra.Command{
		Use:    "sync-keys",
		Hidden: true,
		Short:  "Re-encrypt encrypt keys for all linked public keys",
		Long:   formatLong(fmt.Sprintf("%s encrypt keys for all linked public keys", common.Keyword("Re-encrypt"))),
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cc := initCharmClient()
			return cc.SyncEncryptKeys()
		},
	}
)
