package main

import (
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/charm/client"
	"github.com/charmbracelet/charm/cmd"
	"github.com/charmbracelet/charm/ui"
	"github.com/charmbracelet/charm/ui/common"
	"github.com/spf13/cobra"
)

var (
	Version   = ""
	CommitSHA = ""

	rootCmd = &cobra.Command{
		Use:                   "charm",
		Short:                 "Do Charm stuff",
		Long:                  common.FormatLong(fmt.Sprintf("Do %s stuff. Run without arguments for a TUI or use the sub-commands like a pro.", common.Keyword("Charm"))),
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if common.IsTTY() {
				cfg, err := client.ConfigFromEnv()
				if err != nil {
					log.Fatal(err)
				}

				// Log to file, if set
				if cfg.Logfile != "" {
					f, err := tea.LogToFile(cfg.Logfile, "charm")
					if err != nil {
						return err
					}
					defer f.Close()
				}

				return ui.NewProgram(cfg).Start()
			}

			return cmd.Help()
		},
	}
)

func init() {
	if len(CommitSHA) >= 7 {
		vt := rootCmd.VersionTemplate()
		rootCmd.SetVersionTemplate(vt[:len(vt)-1] + " (" + CommitSHA[0:7] + ")\n")
	}
	if Version == "" {
		Version = "unknown (built from source)"
	}
	rootCmd.Version = Version

	rootCmd.AddCommand(
		cmd.BioCmd,
		cmd.IDCmd,
		cmd.JWTCmd,
		cmd.KeysCmd,
		cmd.KeygenCmd,
		cmd.LinkCmd("charm"),
		cmd.NameCmd,
		cmd.BackupKeysCmd,
		cmd.ImportKeysCmd,
		cmd.KeySyncCmd,
		cmd.CompletionCmd,
		cmd.ServeCmd,
		cmd.KVCmd,
		cmd.FSCmd,
		cmd.CryptCmd,
	)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
