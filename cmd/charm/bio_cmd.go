package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	bioCmd = &cobra.Command{
		Use:    "bio",
		Hidden: true,
		Short:  "",
		Long:   formatLong(""),
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cc := initCharmClient()
			u, err := cc.Bio()
			if err != nil {
				return err
			}

			fmt.Println(u)
			return nil
		},
	}
)
