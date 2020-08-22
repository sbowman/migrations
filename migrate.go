package migrations

import (
	"bufio"
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	// PostgreSQL driver
	_ "github.com/lib/pq"
)

// Direction is the direction to migrate
type Direction string

const (
	// Latest migrates to the latest migration.
	Latest int = -1

	// Up direction.
	Up Direction = "up"

	// Down direction.
	Down Direction = "down"

	// None direction.
	None Direction = "none"
)

var (
	// ErrNameRequired returned if the user failed to supply a name for the
	// migration.
	ErrNameRequired = errors.New("name required")

	// Output defaults to writing to disk.
	IO Reader

	// Matches the Up/Down sections in the SQL migration file
	dirRe = regexp.MustCompile("^#\\s*-{3}\\s*!(.*)\\s*(?:.*)?$")
)

func init() {
	IO = new(DiskReader)
}

// Create a new migration from the template.
func Create(directory string, name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ErrNameRequired
	}

	if err := os.MkdirAll(directory, 0755); err != nil {
		return err
	}

	r := LatestRevision(directory) + 1
	fullname := fmt.Sprintf("%d-%s.sql", r, trimmed)
	path := fmt.Sprintf("%s%c%s", directory, os.PathSeparator, fullname)

	if err := ioutil.WriteFile(path, []byte("# --- !Up\n\n# --- !Down\n\n"), 0644); err != nil {
		return err
	}

	Log.Infof("Created new migration %s", path)
	return nil
}

// Migrate runs the indicated SQL migration files against the database.
//
// Any files that don't have entries in the schema_migrations table will be
// run to bring the database to the indicated version.  If the schema_migrations
// table does not exist, this function will automatically create it.
//
// Indicate the version to roll towards, either forwards or backwards
// (rollback).  By default we roll forwards to the current time, i.e. run all
// the migrations.
//
// If check is true, won't actually run the migrations against Cassandra.
// Instead just simulate the run and report on what would be migrated.
//
// Returns a list of the migrations that were successfully completed, or an
// error if there are problems with the migration.  It is possible for this
// function to return both successful migrations and an error, if some of the
// migrations succeed before an error is encountered.
func Migrate(db *sql.DB, directory string, version int) error {
	if err := CreateSchemaMigrations(db); err != nil {
		return err
	}

	direction := Moving(db, version)
	migrations, err := Available(directory, direction)
	if err != nil {
		return err
	}

	for _, migration := range migrations {
		path := fmt.Sprintf("%s%c%s", directory, os.PathSeparator, migration)

		tx, err := db.Begin()
		if err != nil {
			return err
		}

		if ShouldRun(tx, path, direction, version) {
			Log.Infof("Running migration %s %s", path, direction)
			if err = Run(db, tx, path, direction); err != nil {
				tx.Rollback()
				return err
			}
		}

		if err = tx.Commit(); err != nil {
			return err
		}
	}

	return nil
}

// Rollback a number of migrations.  If steps is less than 2, rolls back the last migration.
func Rollback(db *sql.DB, directory string, steps int) error {
	if steps < 2 {
		steps = 1
	}

	latest, err := LatestMigration(db)
	if err != nil {
		return err
	}

	revision, err := Revision(latest)
	if err != nil {
		return err
	}

	version := revision - steps
	if version < 0 {
		version = 0
	}

	return Migrate(db, directory, version)
}

// Available returns the list of SQL migration paths in order.  If direction is
// Down, returns the migrations in reverse order (migrating down).
func Available(directory string, direction Direction) ([]string, error) {
	files, err := IO.Files(directory)
	if err != nil {
		return nil, fmt.Errorf("invalid migrations directory, %s: %s", directory, err.Error())
	}

	var filenames []string
	for _, name := range files {
		if strings.HasSuffix(name, ".sql") {
			filenames = append(filenames, name)
		}
	}

	if direction == Down {
		sort.Sort(SortDown(filenames))
	} else {
		sort.Sort(SortUp(filenames))
	}

	return filenames, nil
}

// LatestRevision returns the latest revision available from the SQL files in
// the migrations directory.
func LatestRevision(directory string) int {
	migrations, err := Available(directory, Down)
	if err != nil {
		Log.Infof(err.Error())
		return 0
	}

	if len(migrations) == 0 {
		return 0
	}

	// Find a valid filename
	for _, filename := range migrations {
		rev, err := Revision(filename)
		if err != nil {
			Log.Infof("Invalid migration %s: %s", filename, err)
			continue
		}

		return rev
	}

	return 0
}

// Revision extracts the revision number from a migration filename.
func Revision(filename string) (int, error) {
	segments := strings.SplitN(Filename(filename), "-", 2)
	if len(segments) == 1 {
		return 0, fmt.Errorf("invalid migration filename: %s", filename)
	}

	v, err := strconv.Atoi(segments[0])
	if err != nil {
		return 0, err
	}

	return int(v), nil
}

// Filename returns just the filename from the full path.
func Filename(path string) string {
	paths := strings.Split(path, string(os.PathSeparator))
	return paths[len(paths)-1]
}

// Moving determines the direction we're moving to reach the version.
func Moving(db *sql.DB, version int) Direction {
	if version == Latest {
		return Up
	}

	latest, err := LatestMigration(db)
	if err != nil {
		Log.Infof("Unable to get the latest migration: %s", err)
		return None
	}

	if latest == "" {
		return Up
	}

	revision, err := Revision(latest)
	if err != nil {
		Log.Infof("Invalid result from revision: %s", err)
		return None
	}

	if revision < version {
		return Up
	} else if revision > version {
		return Down
	}

	return None
}

// ShouldRun decides if the migration should be applied or removed, based on
// the direction and desired version to reach.
func ShouldRun(tx *sql.Tx, migration string, direction Direction, desiredVersion int) bool {
	version, err := Revision(migration)
	if err != nil {
		Log.Debugf("Unable to determin the revision of %s", migration)
		return false
	}

	switch direction {
	case Up:
		return IsUp(version, desiredVersion) && !IsMigrated(tx, migration)
	case Down:
		return IsDown(version, desiredVersion) && IsMigrated(tx, migration)
	}
	return false
}

// IsUp returns true if the migration must roll up to reach the desired version.
func IsUp(version int, desired int) bool {
	return desired == Latest || version <= desired
}

// IsDown returns true if the migration must rollback to reach the desired
// version.
func IsDown(version int, desired int) bool {
	return version > desired
}

// Run reads the SQL file and applies it to the database.  If successful, mark
// the migration as completed.
func Run(db *sql.DB, tx *sql.Tx, path string, direction Direction) error {
	doc, mods, err := ReadSQL(path, direction)
	if err != nil {
		return err
	}

	// Run the migration outside of the transaction?
	if HasMod(mods, "/notx") {
		_, err = db.Exec(doc)
		if err != nil {
			return err
		}
	} else {
		_, err = tx.Exec(doc)
		if err != nil {
			return err
		}
	}

	if err = Migrated(tx, path, direction); err != nil {
		return err
	}

	return nil
}

// HasMod looks for the migration modification passed in from the SQL.  Mods
// should be indicated like so:
//
//     # --- !Up /notx
//     insert into sample (name) values ('abc');
//
//     # --- !Down /notx
//     delete from sample where name = 'abc';
//
// The only modification supported currently is `/notx`, which runs the SQL
// outside a transaction.
func HasMod(mods []string, value string) bool {
	for _, mod := range mods {
		if strings.EqualFold(mod, value) {
			return true
		}
	}

	return false
}

// ReadSQL reads the migration and filters for the up or down SQL commands.
func ReadSQL(path string, direction Direction) (string, []string, error) {
	f, err := IO.Read(path)
	if err != nil {
		return "", nil, nil
	}

	sqldoc := new(bytes.Buffer)
	parsing := false

	// Collect any modifiers, e.g. /notx, from the SQL direction line
	modifiers := make(map[string]struct{})

	s := bufio.NewScanner(f)
	for s.Scan() {
		found := dirRe.FindStringSubmatch(s.Text())
		if len(found) == 2 {
			mods := strings.Split(found[1], " ")
			dir := strings.ToLower(mods[0])
			mods = mods[1:]

			if Direction(dir) == direction {
				parsing = true

				for _, mod := range mods {
					mod = strings.TrimSpace(mod)
					if mod != "" {
						modifiers[mod] = struct{}{}
					}
				}
				continue
			}

			parsing = false
		} else if parsing {
			sqldoc.Write(s.Bytes())
			sqldoc.WriteRune('\n')
		}
	}

	var mods []string
	for mod := range modifiers {
		mods = append(mods, mod)
	}

	return sqldoc.String(), mods, nil
}

// LatestMigration returns the name of the latest migration run against the
// database.
func LatestMigration(db *sql.DB) (string, error) {
	var latest, migration string

	rows, err := db.Query("select migration from schema_migrations")
	if err != nil {
		return "", err
	}
	defer rows.Close()

	for rows.Next() {
		if err := rows.Scan(&migration); err != nil {
			return "", err
		}

		m, _ := Revision(migration)
		l, _ := Revision(latest)

		if m > l {
			latest = migration
		}
	}

	return latest, nil
}

// IsMigrated checks the migration has been applied to the database, i.e. is it
// in the schema_migrations table?
func IsMigrated(tx *sql.Tx, migration string) bool {
	row := tx.QueryRow("select migration from schema_migrations where migration = $1 limit 1 for update", Filename(migration))
	return row.Scan() != sql.ErrNoRows
}

// Migrated adds or removes the migration record from schema_migrations.
func Migrated(tx *sql.Tx, path string, direction Direction) error {
	var err error
	filename := Filename(path)

	if direction == Down {
		_, err = tx.Exec("delete from schema_migrations where migration = $1", filename)
	} else {
		_, err = tx.Exec("insert into schema_migrations (migration) values ($1)", filename)
	}

	return err
}

// CreateSchemaMigrations creates the schema_migrations table in the database
// if it doesn't already exist.
func CreateSchemaMigrations(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	if MissingSchemaMigrations(tx) {
		Log.Infof("Creating schema_migrations table in the database")
		if _, err := tx.Exec("create table schema_migrations(migration varchar(1024) not null primary key)"); err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

// MissingSchemaMigrations returns true if there is no schema_migrations table
// in the database.
func MissingSchemaMigrations(tx *sql.Tx) bool {
	row := tx.QueryRow("select not(exists(select 1 from pg_catalog.pg_class c " +
		"join pg_catalog.pg_namespace n " +
		"on n.oid = c.relnamespace " +
		"where n.nspname = 'public'and c.relname = 'schema_migrations'))")

	var result bool
	if err := row.Scan(&result); err != nil {
		return true
	}

	return result
}
