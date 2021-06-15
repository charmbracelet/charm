package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// IDCmd is the cobra.Command to print a user's Charm ID.
var IDCmd = &cobra.Command{
	Use:   "id",
	Short: "Print your Charm ID",
	Long:  paragraph("Want to know your " + keyword("Charm ID") + "? You’re in luck, kiddo."),
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cc := initCharmClient(animatedKeygen)
		id, err := cc.ID()
		if err != nil {
			return err
		}

		fmt.Println(id)
		return nil
	},
}
