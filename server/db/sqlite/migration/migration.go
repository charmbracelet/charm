package migration

// Migration is a db migration script.
type Migration struct {
	ID   int
	Name string
	SQL  string
}
