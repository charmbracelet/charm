package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/charm/ui/common"
	"github.com/muesli/reflow/indent"
	"github.com/spf13/cobra"
)

var (
	nameCmd = &cobra.Command{
		Use:     "name [username]",
		Short:   "Username stuff",
		Long:    formatLong("Print or set your " + common.Keyword("username") + ". If the name is already taken, just run it again with a different, cooler name. Basic latin letters and numbers only, 50 characters max."),
		Args:    cobra.RangeArgs(0, 1),
		Example: indent.String("charm name\ncharm name beatrix", indentBy),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc := initCharmClient()
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
				if !charm.ValidateName(n) {
					msg := fmt.Sprintf("%s is invalid.\n\nUsernames must be basic latin letters, numerals, and no more than 50 characters. And no emojis, kid.\n", common.Code(n))
					fmt.Println(formatLong(msg))
					os.Exit(1)
				}
				u, err := cc.SetName(n)
				if err == charm.ErrNameTaken {
					printFormatted(fmt.Sprintf("User name %s is already taken. Try a different, cooler name.\n", common.Code(n)))
					os.Exit(1)
				}
				if err != nil {
					printFormatted(fmt.Sprintf("Welp, there’s been an error. %s", common.Subtle(err.Error())))
					return err
				}

				printFormatted(fmt.Sprintf("OK! Your new username is %s", common.Code(u.Name)))
				return nil
			}
		},
	}
)
