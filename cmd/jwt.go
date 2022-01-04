package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// JWTCmd is the cobra.Command that prints a user's JWT token.
var JWTCmd = &cobra.Command{
	Use:   "jwt",
	Short: "Print a JWT",
	Long:  paragraph(keyword("JSON Web Tokens") + " are a way to authenticate to different services that utilize your Charm account. Use " + code("jwt") + " to get one for your account."),
	Args:  cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cc := initCharmClient(animatedKeygen)
		jwt, err := cc.JWT(args...)
		if err != nil {
			return err
		}

		fmt.Printf("%s\n", jwt)
		return nil
	},
}
