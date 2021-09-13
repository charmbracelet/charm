package cmd

import (
	"fmt"

	"github.com/charmbracelet/charm/server"
	"github.com/charmbracelet/keygen"
	"github.com/spf13/cobra"
)

var (
	serverHTTPPort int
	serverSSHPort  int
	serverDataDir  string

	//ServeCmd is the cobra.Command to self-host the Charm Cloud.
	ServeCmd = &cobra.Command{
		Use:     "serve",
		Aliases: []string{"server"},
		Hidden:  false,
		Short:   "Start a self-hosted Charm Cloud server.",
		Long:    paragraph("Start the SSH and HTTP servers needed to power a SQLite-backed Charm Cloud."),
		Args:    cobra.NoArgs,
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
			kp, err := keygen.NewWithWrite(sp, "charm_server", []byte(""), keygen.RSA)
			if err != nil {
				return err
			}
			cfg = cfg.WithKeys(kp.PublicKey, kp.PrivateKeyPEM)
			s, err := server.NewServer(cfg)
			if err != nil {
				return err
			}
			s.Start()
			return nil
		},
	}
)

func init() {
	ServeCmd.Flags().IntVar(&serverHTTPPort, "http-port", 0, "HTTP port to listen on")
	ServeCmd.Flags().IntVar(&serverSSHPort, "ssh-port", 0, "SSH port to listen on")
	ServeCmd.Flags().StringVar(&serverDataDir, "data-dir", "", "Directory to store SQLite db, SSH keys and file data")
}
