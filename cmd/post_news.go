package cmd

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/charm/server"
	"github.com/charmbracelet/keygen"
	"github.com/spf13/cobra"
)

var newsSubject string
var newsTagList string

var (
	//PostNewsCmd is the cobra.Command to self-host the Charm Cloud.
	PostNewsCmd = &cobra.Command{
		Use:    "post-news",
		Hidden: true,
		Short:  "Post news to the self-hosted Charm server.",
		Args:   cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := server.DefaultConfig()
			if serverDataDir != "" {
				cfg.DataDir = serverDataDir
			}
			sp := filepath.Join(cfg.DataDir, ".ssh")
			kp, err := keygen.NewWithWrite(sp, "charm_server", []byte(""), keygen.Ed25519)
			if err != nil {
				return err
			}
			cfg = cfg.WithKeys(kp.PublicKey, kp.PrivateKeyPEM)
			s, err := server.NewServer(cfg)
			if err != nil {
				return err
			}
			if newsSubject == "" {
				newsSubject = args[0]
			}
			ts := strings.Split(newsTagList, ",")
			d, err := os.ReadFile(args[0])
			if err != nil {
				return err
			}
			err = s.Config.DB.PostNews(newsSubject, string(d), ts)
			if err != nil {
				return err
			}
			return nil
		},
	}
)

func init() {
	PostNewsCmd.Flags().StringVarP(&newsSubject, "subject", "s", "", "Subject for news post")
	PostNewsCmd.Flags().StringVarP(&newsTagList, "tags", "t", "server", "Tags for news post, comma separated")
	PostNewsCmd.Flags().StringVarP(&serverDataDir, "data-dir", "", "", "Directory to store SQLite db, SSH keys and file data")
}
