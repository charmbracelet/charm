package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/charm/client"
	charm "github.com/charmbracelet/charm/proto"
	"github.com/muesli/reflow/indent"
	"github.com/spf13/cobra"
)

// NameCmd is the cobra.Command to print or set a username.
var NameCmd = &cobra.Command{
	Use:     "name [username]",
	Short:   "Username stuff",
	Long:    paragraph("Print or set your " + keyword("username") + ". If the name is already taken, just run it again with a different, cooler name. Basic latin letters and numbers only, 50 characters max."),
	Args:    cobra.RangeArgs(0, 1),
	Example: indent.String("charm name\ncharm name beatrix", 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cc := initCharmClient(animatedKeygen)
		switch len(args) {
		case 0:
			u, err := cc.Bio()
			if err != nil {
				return err
			}

			fmt.Println(u.Name)
			return nil
		default:
			n := args[0]
			if !client.ValidateName(n) {
				msg := fmt.Sprintf("%s is invalid.\n\nUsernames must be basic latin letters, numerals, and no more than 50 characters. And no emojis, kid.\n", code(n))
				fmt.Println(paragraph(msg))
				os.Exit(1)
			}
			u, err := cc.SetName(n)
			if err == charm.ErrNameTaken {
				paragraph(fmt.Sprintf("User name %s is already taken. Try a different, cooler name.\n", code(n)))
				os.Exit(1)
			}
			if err != nil {
				paragraph(fmt.Sprintf("Welp, there’s been an error. %s", subtle(err.Error())))
				return err
			}

			paragraph(fmt.Sprintf("OK! Your new username is %s", code(u.Name)))
			return nil
		}
	},
}
