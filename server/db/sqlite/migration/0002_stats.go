package migration

// Migration0002 adds new columns to the stats table.
var Migration0002 = Migration{
	ID:   2,
	Name: "metrics",
	SQL: `
ALTER TABLE stats
ADD COLUMN fs_files_read integer NOT NULL DEFAULT 0,
ADD COLUMN fs_files_written integer NOT NULL DEFAULT 0;
	`,
}
