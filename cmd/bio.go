package cmd

import (
	"fmt"

	"github.com/charmbracelet/charm/ui/common"
	"github.com/spf13/cobra"
)

var (
	// BioCmd is the cobra.Command to return a user's bio JSON result.
	BioCmd = &cobra.Command{
		Use:    "bio",
		Hidden: true,
		Short:  "",
		Long:   common.FormatLong(""),
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cc := initCharmClient(animatedKeygen)
			u, err := cc.Bio()
			if err != nil {
				return err
			}

			fmt.Println(u)
			return nil
		},
	}
)
