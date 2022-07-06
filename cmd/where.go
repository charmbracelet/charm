package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var WhereCmd = &cobra.Command{
	Use:   "where",
	Short: "Find where your cloud.charm.sh folder resides on your machine",
	Long:  paragraph("Find the absolute path to your charm keys, databases, etc."),
	RunE: func(cmd *cobra.Command, args []string) error {
		cc := initCharmClient()
		path, err := cc.DataPath()
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), path)
		return nil
	},
}
