package main

import (
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/charm/ui"
	"github.com/charmbracelet/charm/ui/common"
	"github.com/mattn/go-isatty"
	"github.com/muesli/reflow/indent"
	"github.com/muesli/reflow/wordwrap"
	"github.com/spf13/cobra"
)

const (
	wrapAt   = 78
	indentBy = 2
)

func formatLong(s string) string {
	return indent.String(wordwrap.String("\n"+s, wrapAt), indentBy)
}

func printFormatted(s string) {
	fmt.Println(formatLong(s + "\n"))
}

func isTTY() bool {
	return isatty.IsTerminal(os.Stdout.Fd())
}

var (
	Version   = ""
	CommitSHA = ""

	simpleOutput bool
	randomart    bool

	rootCmd = &cobra.Command{
		Use:                   "charm",
		Short:                 "Do Charm stuff",
		Long:                  formatLong(fmt.Sprintf("Do %s stuff. Run without arguments for a TUI or use the sub-commands like a pro.", common.Keyword("Charm"))),
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if isTTY() {
				cfg := getCharmConfig()

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

func getCharmConfig() *charm.Config {
	cfg, err := charm.ConfigFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	return cfg
}

func initCharmClient() *charm.Client {
	cfg := getCharmConfig()
	cc, err := charm.NewClient(cfg)
	if err == charm.ErrMissingSSHAuth {
		printFormatted("We were’t able to authenticate via SSH, which means there’s likely a problem with your key.\n\nYou can generate SSH keys by running " + common.Code("charm keygen") + ". You can also set the environment variable " + common.Code("CHARM_SSH_KEY_PATH") + " to point to a specific private key, or use " + common.Code("-i") + "specifify a location.")
		os.Exit(1)
	} else if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return cc
}

func init() {
	if len(CommitSHA) >= 7 {
		vt := rootCmd.VersionTemplate()
		rootCmd.SetVersionTemplate(vt[:len(vt)-1] + " (" + CommitSHA[0:7] + ")\n")
	}
	if Version == "" {
		Version = "unknown (built from source)"
	}
	rootCmd.Version = Version

	importKeysCmd.Flags().BoolVarP(&forceImportOverwrite, "force-overwrite", "f", false, "overwrite if keys exist; don’t prompt for input")
	keysCmd.Flags().BoolVarP(&simpleOutput, "simple", "s", false, "simple, non-interactive output (good for scripts)")
	keysCmd.Flags().BoolVarP(&randomart, "randomart", "r", false, "print SSH 5.1 randomart for each key (the Drunken Bishop algorithm)")

	rootCmd.AddCommand(
		bioCmd,
		idCmd,
		jwtCmd,
		keysCmd,
		keygenCmd,
		linkCmd,
		nameCmd,
		encryptCmd,
		decryptCmd,
		backupKeysCmd,
		importKeysCmd,
		keySyncCmd,
		completionCmd,
	)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
