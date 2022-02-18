package cmd

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/charmbracelet/charm/server"
	"github.com/charmbracelet/charm/server/db/sqlite"
	"github.com/charmbracelet/charm/server/db/sqlite/migration"
	"github.com/spf13/cobra"
	_ "modernc.org/sqlite"
)

var (
	ServeMigrationCmd = &cobra.Command{
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
			log.Printf("Opening SQLite db: %s\n", dp)
			db, err := sql.Open("sqlite", dp+sqlite.DbOptions)
			if err != nil {
				return err
			}
			for _, m := range []migration.Migration{
				migration.Migration0001,
			} {
				log.Printf("Running migration: %04d %s\n", m.ID, m.Name)
				tx, err := db.Begin()
				if err != nil {
					return err
				}
				_, err = tx.Exec(m.Sql)
				if err != nil {
					return err
				}
				err = tx.Commit()
				if err != nil {
					return err
				}
				return nil
			}
			if err != nil {
				return err
			}
			return nil
		},
	}
)
