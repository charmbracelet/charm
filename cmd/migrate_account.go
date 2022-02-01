package cmd

import (
	"log"

	"github.com/charmbracelet/charm/client"
	"github.com/charmbracelet/charm/proto"
	"github.com/spf13/cobra"
)

// MigrateAccountCmd is a command to convert your legacy RSA SSH keys to the
// new Ed25519 standard keys
var MigrateAccountCmd = &cobra.Command{
	Use:    "migrate-account",
	Hidden: true,
	Short:  "",
	Long:   "",
	Args:   cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
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
		lh := &linkHandler{linkChan: make(chan string)}
		go func() {
			rsaClient.LinkGen(lh)
		}()
		tok := <-lh.linkChan
		ed25519Client.Link(lh, tok)
		err = rsaClient.SyncEncryptKeys()
		if err != nil {
			return err
		}
		err = ed25519Client.SyncEncryptKeys()
		if err != nil {
		}
		return nil
	},
}

type linkHandler struct {
	linkChan chan string
}

func (lh *linkHandler) TokenCreated(l *proto.Link) {
	// log.Printf("token created %v", l)
	lh.linkChan <- string(l.Token)
}

func (lh *linkHandler) TokenSent(l *proto.Link) {
	// log.Printf("token sent %v", l)
}

func (lh *linkHandler) ValidToken(l *proto.Link) {
	// log.Printf("valid token %v", l)
}

func (lh *linkHandler) InvalidToken(l *proto.Link) {
	// log.Printf("invalid token %v", l)
}

func (lh *linkHandler) Request(l *proto.Link) bool {
	// log.Printf("request %v", l)
	return true
}

func (lh *linkHandler) RequestDenied(l *proto.Link) {
	// log.Printf("request denied %v", l)
}

func (lh *linkHandler) SameUser(l *proto.Link) {
	log.Println("Success! Looks like you've done this already.")
}

func (lh *linkHandler) Success(l *proto.Link) {
	log.Println("Success! You're good to go.")
}

func (lh *linkHandler) Timeout(l *proto.Link) {
	log.Println("Timeout")
}

func (lh linkHandler) Error(l *proto.Link) {
	log.Printf("Error %v", l)
}
