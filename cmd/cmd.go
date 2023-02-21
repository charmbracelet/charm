// Package cmd implements the Cobra commands for the charm CLI.
package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/log"

	"github.com/charmbracelet/charm/client"
	charm "github.com/charmbracelet/charm/proto"
	"github.com/charmbracelet/charm/ui/common"
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

func initCharmClient() *client.Client {
	cfg := getCharmConfig()
	cc, err := client.NewClient(cfg)
	if err == charm.ErrMissingSSHAuth {
		printFormatted("We were’t able to authenticate via SSH, which means there’s likely a problem with your key.\n\nYou can generate SSH keys by running " + code("charm keygen") + ". You can also set the environment variable " + code("CHARM_SSH_KEY_PATH") + " to point to a specific private key, or use " + code("-i") + "specifify a location.")
		os.Exit(1)
	} else if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return cc
}
