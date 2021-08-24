package migrations

import (
	"database/sql"
	"fmt"
	"os"
)

// MigrateUnsafe runs like Migrate, but does not run the migrations in a transaction.
//
// If your database server supports running DDL statements in tranasaction, you should be calling
// Migrate.  This should only be used for databases that don't support DDL statements in a
// transaction.
//
// The Migrate function runs each migration in a database transaction.  If the migration fails part
// of the way through, everything rolls back and the database is in the same state as it was before
// the migration was run.  Migrations run using MigrateUnsafe will run without the protection of a
// database transaction.  With MigrateUnsafe, if a migration containing multiple SQL commands fails
// part of the way through the migration, it leaves the database with the migration partially
// applied, i.e. some commands succeeded, while the erroneous SQL and later statements were not
// applied.  This likely requries manual intervention to roll out the commands that did succeeed by
// hand, toclean up and restore to the state it was in before the migration was run, so it can be
// applied again when the issue is corrected.
//
// If you use MigrateUnsafe, you should also write smaller migrations with fewer SQL commands.
// This way, if a migration partially fails, manually undoing the successful portion is an easier
// prospect.
//
// Some databases do not support DDL statements in a transaction, and trying to run them in a
// transaction can cause a variety of issues.
func MigrateUnsafe(db *sql.DB, directory string, version int) error {
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

		if ShouldRunNoTx(db, path, direction, version) {
			SQL, _, err := ReadSQL(path, direction)
			if err != nil {
				return err
			}

			Log.Infof("Running migration %s %s", path, direction)

			_, err = db.Exec(string(SQL))
			if err != nil {
				return err
			}

			if err = MigratedNoTx(db, path, direction); err != nil {
				return err
			}
		}
	}

	return nil
}

// ShouldRunNoTx decides if the migration should be applied or removed, based on
// the direction and desired version to reach.
//
// This version is like ShouldRun, but doesn't use a transaction.
func ShouldRunNoTx(db *sql.DB, migration string, direction Direction, desiredVersion int) bool {
	version, err := Revision(migration)
	if err != nil {
		Log.Debugf("Unable to determine the revision of %s", migration)
		return false
	}

	switch direction {
	case Up:
		return IsUp(version, desiredVersion) && !IsMigratedNoTx(db, migration)
	case Down:
		return IsDown(version, desiredVersion) && IsMigratedNoTx(db, migration)
	}
	return false
}

// IsMigratedNoTx checks the migration has been applied to the database, i.e. is it
// in the schema_migrations table?
//
// This version is like IsMigrated, but doesn't use a transaction.
func IsMigratedNoTx(db *sql.DB, migration string) bool {
	row := db.QueryRow("select migration from schema_migrations where migration = $1 limit 1 for update", Filename(migration))
	return row.Scan() != sql.ErrNoRows
}

// MigratedNoTx adds or removes the migration record from schema_migrations.
//
// This version is like Migrated, but doesn't use a transaction.
func MigratedNoTx(db *sql.DB, path string, direction Direction) error {
	var err error
	filename := Filename(path)

	if direction == Down {
		_, err = db.Exec("delete from schema_migrations where migration = $1", filename)
	} else {
		_, err = db.Exec("insert into schema_migrations (migration) values ($1)", filename)
	}

	return err
}
