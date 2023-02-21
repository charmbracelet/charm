package cmd

import (
	"fmt"

	"github.com/charmbracelet/log"

	"github.com/charmbracelet/charm/client"
	"github.com/charmbracelet/charm/proto"
	"github.com/spf13/cobra"
)

// MigrateAccountCmd is a command to convert your legacy RSA SSH keys to the
// new Ed25519 standard keys.
var (
	verbose   bool
	linkError bool

	MigrateAccountCmd = &cobra.Command{
		Use:    "migrate-account",
		Hidden: true,
		Short:  "",
		Long:   "",
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Migrating account...")
			rcfg, err := client.ConfigFromEnv()
			if err != nil {
				return err
			}
			rcfg.KeyType = "rsa"
			rsaClient, err := client.NewClient(rcfg)
			if err != nil {
				return err
			}

			ecfg, err := client.ConfigFromEnv()
			if err != nil {
				return err
			}
			ecfg.KeyType = "ed25519"
			ed25519Client, err := client.NewClient(ecfg)
			if err != nil {
				return err
			}

			lc := make(chan string)
			go func() {
				lh := &linkHandler{desc: "link-gen", linkChan: lc}
				_ = rsaClient.LinkGen(lh)
			}()
			tok := <-lc
			lh := &linkHandler{desc: "link-request", linkChan: lc}
			_ = ed25519Client.Link(lh, tok)
			if verbose {
				log.Info("link-gen sync encrypt keys")
			}
			err = rsaClient.SyncEncryptKeys()
			if err != nil {
				if verbose {
					log.Info("link-gen sync encrypt keys failed")
				} else {
					printError()
				}
				return err
			}
			if verbose {
				log.Info("link-request sync encrypt keys")
			}
			err = ed25519Client.SyncEncryptKeys()
			if err != nil {
				if verbose {
					log.Info("link-request sync encrypt keys failed")
				} else {
					printError()
				}
				return err
			}
			if !linkError {
				fmt.Println("Account migrated! You're good to go.")
			} else {
				printError()
			}
			return nil
		},
	}
)

type linkHandler struct {
	desc     string
	linkChan chan string
}

func (lh *linkHandler) TokenCreated(l *proto.Link) {
	lh.printDebug("token created", l)
	lh.linkChan <- string(l.Token)
	lh.printDebug("token created sent to chan", l)
}

func (lh *linkHandler) TokenSent(l *proto.Link) {
	lh.printDebug("token sent", l)
}

func (lh *linkHandler) ValidToken(l *proto.Link) {
	lh.printDebug("valid token", l)
}

func (lh *linkHandler) InvalidToken(l *proto.Link) {
	lh.printDebug("invalid token", l)
}

func (lh *linkHandler) Request(l *proto.Link) bool {
	lh.printDebug("request", l)
	return true
}

func (lh *linkHandler) RequestDenied(l *proto.Link) {
	lh.printDebug("request denied", l)
}

func (lh *linkHandler) SameUser(l *proto.Link) {
	lh.printDebug("same user", l)
}

func (lh *linkHandler) Success(l *proto.Link) {
	lh.printDebug("success", l)
}

func (lh *linkHandler) Timeout(l *proto.Link) {
	lh.printDebug("timeout", l)
}

func (lh linkHandler) Error(l *proto.Link) {
	linkError = true
	lh.printDebug("error", l)
	if !verbose {
		printError()
	}
}

func (lh *linkHandler) printDebug(msg string, l *proto.Link) {
	if verbose {
		log.Info("%s %s:\t%v\n", lh.desc, msg, l)
	}
}

func printError() {
	fmt.Println("\nThere was an error migrating your account. Please re-run with the -v argument `charm migrate-account -v` and join our slack at https://charm.sh/slack to help debug the issue. Sorry about that, we'll try to figure it out!")
}

func init() {
	MigrateAccountCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "print debug output")
}
