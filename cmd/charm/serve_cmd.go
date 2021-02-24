package main

import (
	"fmt"

	"github.com/charmbracelet/charm/keygen"
	"github.com/charmbracelet/charm/server"
	"github.com/spf13/cobra"
)

var (
	serverHTTPPort int
	serverSSHPort  int
	serverDataDir  string

	serveCmd = &cobra.Command{
		Use:    "serve",
		Hidden: false,
		Short:  "Start a self-hosted Charm Cloud server.",
		Long:   formatLong(fmt.Sprintf("Start the SSH and HTTP servers needed to power a SQLite backed Charm Cloud.")),
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := server.DefaultConfig()
			if serverHTTPPort != 0 {
				cfg.HTTPPort = serverHTTPPort
			}
			if serverSSHPort != 0 {
				cfg.SSHPort = serverSSHPort
			}
			if serverDataDir != "" {
				cfg.DataDir = serverDataDir
			}
			sp := fmt.Sprintf("%s/.ssh", cfg.DataDir)
			kp, err := keygen.NewSSHKeyPair(sp, "charm_server", []byte(""), "rsa")
			if err != nil {
				return err
			}
			cfg = cfg.WithKeys(kp.PublicKey, kp.PrivateKeyPEM)
			s := server.NewServer(cfg)
			s.Start()
			return nil
		},
	}
)

func init() {
	serveCmd.Flags().IntVar(&serverHTTPPort, "http-port", 0, "HTTP port to listen on.")
	serveCmd.Flags().IntVar(&serverSSHPort, "ssh-port", 0, "SSH port to listen on.")
	serveCmd.Flags().StringVar(&serverDataDir, "data-dir", "", "Directory to store SQLite db, SSH keys and file data.")
}
