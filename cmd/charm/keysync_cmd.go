package main

import "github.com/spf13/cobra"

var (
	keySyncCmd = &cobra.Command{
		Use:    "sync-keys",
		Hidden: true,
		Short:  "Re-encrypt encrypt keys for all linked public keys",
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cc := initCharmClient()
			return cc.SyncEncryptKeys()
		},
	}
)
