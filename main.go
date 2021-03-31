package main

import (
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/charm/client"
	"github.com/charmbracelet/charm/keygen"
	charm "github.com/charmbracelet/charm/proto"
	"github.com/charmbracelet/charm/ui"
	"github.com/charmbracelet/charm/ui/common"
	keygenTUI "github.com/charmbracelet/charm/ui/keygen"
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

func getCharmConfig() *client.Config {
	cfg, err := client.ConfigFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	return cfg
}

type keygenSetting int

const (
	noKeygen       keygenSetting = iota // don't generate keys
	animatedKeygen                      // generate keys; if input is a TTY show progress with a spinner
	silentKeygen                        // generate keys silently
)

func initCharmClient(kg keygenSetting) *client.Client {
	cfg := getCharmConfig()
	cc, err := client.NewClient(cfg)
	if err == charm.ErrMissingSSHAuth {

		if kg != noKeygen {
			keygenError := "Uh oh. We tried to generate a new pair of keys for your " + common.Keyword("Charm Account") + " but we hit a snag:\n\n"

			if isatty.IsTerminal(os.Stdout.Fd()) {
				// Generate	keys, using Bubble Tea for feedback
				err := keygenTUI.NewProgram(false).Start()
				if err != nil {
					printFormatted(keygenError + err.Error())
					os.Exit(1)
				}
			} else {
				// Generate keys
				dp, err := client.DataPath()
				if err != nil {
					printFormatted(keygenError + err.Error())
					os.Exit(1)
				}
				_, err = keygen.NewSSHKeyPair(dp, "charm", []byte(""), "rsa")
				if err != nil {
					printFormatted(keygenError + err.Error())
					os.Exit(1)
				}
			}
			// Now try again
			return initCharmClient(noKeygen)
		}

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
		serveCmd,
		kvCmd,
		fsCmd,
	)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
