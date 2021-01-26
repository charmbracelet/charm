package main

import (
	"github.com/charmbracelet/charm/keygen"
	"github.com/charmbracelet/charm/server"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "server",
	Short: "Charm Cloud server",
	Long:  "Charm Cloud SSH and HTTP server for identity and storage services.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func main() {
	kp, err := keygen.NewSSHKeyPair(".ssh", "charm_server", []byte(""), "rsa")
	if err != nil {
		panic(err)
	}
	cfg := server.DefaultConfig().WithKeys(kp.PublicKey, kp.PrivateKeyPEM)
	s := server.NewServer(cfg)
	s.Start()
}
