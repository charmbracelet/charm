package main

import (
	"fmt"

	"github.com/charmbracelet/charm/ui/common"
	"github.com/spf13/cobra"
)

var (
	idCmd = &cobra.Command{
		Use:   "id",
		Short: "Print your Charm ID",
		Long:  formatLong("Want to know your " + common.Keyword("Charm ID") + "? You’re in luck, kiddo."),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cc := initCharmClient()
			id, err := cc.ID()
			if err != nil {
				return err
			}

			fmt.Println(id)
			return nil
		},
	}
)
