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

	"github.com/sbowman/glog"
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
	//log   = logging.MustGetLogger("migrations")
	dirRe = regexp.MustCompile("^#\\s*\\-{3}\\s*!(.*)\\s*(?:.*)?$")
)

// Create a new migration from the template.
func Create(directory string, name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return errors.New("Name required.")
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

	println("Created new migration " + path)
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
			glog.Infof("Running migration %s %s", path, direction)
			if err = Run(tx, path, direction); err != nil {
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

// Available returns the list of SQL migration paths in order.  If direction is
// Down, returns the migrations in reverse order (migrating down).
func Available(directory string, direction Direction) ([]string, error) {
	files, err := ioutil.ReadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("Invalid migrations directory, %s: %s", directory, err.Error())
	}

	var filenames []string
	for _, info := range files {
		if strings.HasSuffix(info.Name(), ".sql") {
			filenames = append(filenames, info.Name())
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
		println(err.Error())
		return 0
	}

	if len(migrations) == 0 {
		return 0
	}

	// Find a valid filename
	for _, filename := range migrations {
		rev, err := Revision(filename)
		if err != nil {
			glog.Warningf("Invalid migration: %s", filename)
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
		return 0, fmt.Errorf("Invalid migration filename: %s", filename)
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
		glog.Errorf("Unable to get latest migration: %s", err)
		return None
	}

	if latest == "" {
		return Up
	}

	revision, err := Revision(latest)
	if err != nil {
		glog.Errorf("Invalid result from revision: %s", err)
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
		glog.Warningf("Unable to determine the revision of %s", migration)
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
func Run(tx *sql.Tx, path string, direction Direction) error {
	doc, err := ReadSQL(path, direction)
	if err != nil {
		return err
	}

	_, err = tx.Exec(doc)
	if err != nil {
		return err
	}

	if err = Migrated(tx, path, direction); err != nil {
		return err
	}

	return nil
}

// ReadSQL reads the migration and filters for the up or down SQL commands.
func ReadSQL(path string, direction Direction) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}

	sql := new(bytes.Buffer)
	parsing := false
	s := bufio.NewScanner(f)
	for s.Scan() {
		found := dirRe.FindStringSubmatch(s.Text())
		if len(found) == 2 {
			parsing = Direction(strings.ToLower(found[1])) == direction
		} else if parsing {
			sql.Write(s.Bytes())
			sql.WriteRune('\n')
		}
	}

	return sql.String(), nil
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
		glog.Info("Creating schema_migrations table in database.")
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
