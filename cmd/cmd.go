// Package cmd implements the Cobra commands for the charm CLI.
package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/charmbracelet/charm/client"
	charm "github.com/charmbracelet/charm/proto"
	"github.com/charmbracelet/charm/ui/common"
	keygenTUI "github.com/charmbracelet/charm/ui/keygen"
	"github.com/charmbracelet/keygen"
	"github.com/mattn/go-isatty"
)

type keygenSetting int

const (
	noKeygen       keygenSetting = iota // don't generate keys
	animatedKeygen                      // generate keys; if input is a TTY show progress with a spinner
	silentKeygen                        // generate keys silently
)

var (
	styles    = common.DefaultStyles()
	paragraph = styles.Paragraph.Render
	keyword   = styles.Keyword.Render
	code      = styles.Code.Render
	subtle    = styles.Subtle.Render
)

func printFormatted(s string) {
	fmt.Println(paragraph(s) + "\n")
}

func getCharmConfig() *client.Config {
	cfg, err := client.ConfigFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	return cfg
}

func initCharmClient(kg keygenSetting) *client.Client {
	cfg := getCharmConfig()
	cc, err := client.NewClient(cfg)
	if err == charm.ErrMissingSSHAuth {

		if kg != noKeygen {
			keygenError := "Uh oh. We tried to generate a new pair of keys for your " + keyword("Charm Account") + " but we hit a snag:\n\n"

			if isatty.IsTerminal(os.Stdout.Fd()) {
				// Generate	keys, using Bubble Tea for feedback
				err := keygenTUI.NewProgram(cfg.Host, false).Start()
				if err != nil {
					printFormatted(keygenError + err.Error())
					os.Exit(1)
				}
			} else {
				// Generate keys
				dp, err := client.DataPath(cfg.Host)
				if err != nil {
					printFormatted(keygenError + err.Error())
					os.Exit(1)
				}
				_, err = keygen.NewWithWrite(dp, "charm", []byte(""), cfg.KeygenType())
				if err != nil {
					printFormatted(keygenError + err.Error())
					os.Exit(1)
				}
			}
			// Now try again
			return initCharmClient(noKeygen)
		}

		printFormatted("We were’t able to authenticate via SSH, which means there’s likely a problem with your key.\n\nYou can generate SSH keys by running " + code("charm keygen") + ". You can also set the environment variable " + code("CHARM_SSH_KEY_PATH") + " to point to a specific private key, or use " + code("-i") + " to specify a location.")
		os.Exit(1)
	} else if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return cc
}
