package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// BioCmd is the cobra.Command to return a user's bio JSON result.
var BioCmd = &cobra.Command{
	Use:    "bio",
	Hidden: true,
	Short:  "",
	Long:   "",
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
