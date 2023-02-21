package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/charmbracelet/log"

	"github.com/charmbracelet/charm/client"
	"github.com/charmbracelet/charm/cmd"
	"github.com/charmbracelet/charm/ui"
	"github.com/charmbracelet/charm/ui/common"
	mcobra "github.com/muesli/mango-cobra"
	"github.com/muesli/roff"
	"github.com/spf13/cobra"
)

var (
	// Version is the version of the charm CLI.
	Version = ""
	// CommitSHA is the commit SHA of the charm CLI.
	CommitSHA = ""

	styles = common.DefaultStyles()

	rootCmd = &cobra.Command{
		Use:                   "charm",
		Short:                 "Do Charm stuff",
		Long:                  styles.Paragraph.Render(fmt.Sprintf("Do %s stuff. Run without arguments for a TUI or use the sub-commands like a pro.", styles.Keyword.Render("Charm"))),
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if common.IsTTY() {
				cfg, err := client.ConfigFromEnv()
				if err != nil {
					log.Fatal(err)
				}

				// Log to file, if set
				if cfg.Logfile != "" {
					f, err := os.OpenFile(cfg.Logfile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
					if err != nil {
						return err
					}
					if cfg.Debug {
						log.SetLevel(log.DebugLevel)
					}
					log.SetOutput(f)
					log.SetPrefix("charm")

					defer f.Close() // nolint: errcheck
				}

				p := ui.NewProgram(cfg)
				if _, err := p.Run(); err != nil {
					return err
				}
			}

			return cmd.Help()
		},
	}

	manCmd = &cobra.Command{
		Use:    "man",
		Short:  "Generate man pages",
		Args:   cobra.NoArgs,
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			manPage, err := mcobra.NewManPage(1, rootCmd) //.
			if err != nil {
				return err
			}

			manPage = manPage.WithSection("Copyright", "(C) 2021-2022 Charmbracelet, Inc.\n"+
				"Released under MIT license.")
			fmt.Println(manPage.Build(roff.NewDocument()))
			return nil
		},
	}
)

func init() {
	if len(CommitSHA) >= 7 {
		vt := rootCmd.VersionTemplate()
		rootCmd.SetVersionTemplate(vt[:len(vt)-1] + " (" + CommitSHA[0:7] + ")\n")
	}
	if Version == "" {
		if info, ok := debug.ReadBuildInfo(); ok && info.Main.Sum != "" {
			Version = info.Main.Version
		} else {
			Version = "unknown (built from source)"
		}
	}
	rootCmd.Version = Version
	rootCmd.CompletionOptions.HiddenDefaultCmd = true

	rootCmd.AddCommand(
		cmd.BioCmd,
		cmd.IDCmd,
		cmd.JWTCmd,
		cmd.KeysCmd,
		cmd.LinkCmd("charm"),
		cmd.NameCmd,
		cmd.BackupKeysCmd,
		cmd.ImportKeysCmd,
		cmd.KeySyncCmd,
		cmd.CompletionCmd,
		cmd.ServeCmd,
		cmd.PostNewsCmd,
		cmd.KVCmd,
		cmd.FSCmd,
		cmd.CryptCmd,
		cmd.MigrateAccountCmd,
		cmd.WhereCmd,
		manCmd,
	)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
