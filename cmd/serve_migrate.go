package cmd

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"

	"github.com/charmbracelet/charm/server"
	"github.com/charmbracelet/charm/server/db/sqlite"
	"github.com/charmbracelet/charm/server/db/sqlite/migration"
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
		dp := filepath.Join(cfg.DataDir, "db", sqlite.DbName)
		_, err := os.Stat(dp)
		if err != nil {
			return fmt.Errorf("database does not exist: %s", err)
		}
		db := sqlite.NewDB(dp)
		for _, m := range []migration.Migration{
			migration.Migration0001,
		} {
			log.Print("Running migration", "id", fmt.Sprintf("%04d", m.ID), "name", m.Name)
			err = db.WrapTransaction(func(tx *sql.Tx) error {
				_, err := tx.Exec(m.SQL)
				if err != nil {
					return err
				}
				return nil
			})
			if err != nil {
				break
			}
		}
		if err != nil {
			return err
		}
		return nil
	},
}
