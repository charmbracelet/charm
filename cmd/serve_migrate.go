package cmd

import (
	"path/filepath"

	"github.com/charmbracelet/charm/server"
	"github.com/charmbracelet/charm/server/db/sqlite"
	"github.com/charmbracelet/charm/server/storage"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"

	_ "modernc.org/sqlite" // sqlite driver
)

// ServeMigrationCmd migrate server db.
var ServeMigrationCmd = &cobra.Command{
	Use:     "migrate",
	Aliases: []string{"migration"},
	Hidden:  true,
	Short:   "Run the server migration tool.",
	Long:    paragraph("Run the server migration tool to migrate the database."),
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := server.DefaultConfig()
		dp := filepath.Join(cfg.DataDir, "db")
		err := storage.EnsureDir(dp, 0o700)
		if err != nil {
			log.Fatal("could not init sqlite path", "err", err)
		}
		return sqlite.NewDB(filepath.Join(dp, sqlite.DbName)).Migrate()
	},
}
