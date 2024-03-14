package migration

import (
	"fmt"
	"time"

	"github.com/charmbracelet/log"
)

// Migrations is a list of all migrations.
// The migrations must be in sequence starting from 1.
var Migrations = []Migration{
	Migration0001,
	Migration0002,
}

// Migration is a db migration script.
type Migration struct {
	Version int
	Name    string
	SQL     string
}

type Version struct {
	Version     int
	Name        *string
	CompletedAt *time.Time
	ErrorAt     *time.Time
	Comment     *string
	CreatedAt   *time.Time
	UpdatedAt   *time.Time
}

func (v Version) String() string {
	return fmt.Sprintf(
		"Version: %d, Name: %s, CompletedAt: %s, ErrorAt: %s, Comment: %s, CreatedAt: %s, UpdatedAt: %s",
		v.Version,
		safeString(v.Name),
		safeTime(v.CompletedAt),
		safeTime(v.ErrorAt),
		safeString(v.Comment),
		safeTime(v.CreatedAt),
		safeTime(v.UpdatedAt),
	)
}
func safeString(s *string) string {
	if s != nil {
		return *s
	}
	return "nil"
}
func safeTime(t *time.Time) string {
	if t != nil {
		return t.Format(time.RFC3339)
	}
	return "nil"
}

// Validate validates the migration sequence.
// It returns an error if the sequence is not valid.
// Each migration must have a unique version number and
// the version numbers must be in sequence starting from 1.
func Validate() error {
	log.Info("validating migrations")
	// later, this could be changed to ensure all versions are sequential starting from the first item in the array
	// this would remove the requirement to have all versions starting from 1.
	// this would allow to 'prune' or 'compact' previous versions in some way while continuing the general version scheme.
	for i, m := range Migrations {
		if i+1 != m.Version {
			log.Error("migration is not in sequence", "expected", i+1, "actual", m.Version, "migration", m)
			return fmt.Errorf("migration %d is not in sequence, expected %d, name %s", m.Version, i+1, m.Name)
		}
	}
	return nil
}
