package migrations

import "database/sql"

// Migrate runs the indicated SQL migration files against the database.
//
// This function is provided for backwards compatibility with the older migrations/v1 package.  The
// new v2 approach is to call `Apply()` with any options supplied separately.
//
// Any files that don't have entries in the schema_migrations table will be run to bring the
// database to the indicated version.  If the schema_migrations table does not exist, this function
// will automatically create it.
//
// Indicate the version to roll towards, either forwards or backwards (rollback).  By default, we
// roll forwards to the current time, i.e. run all the migrations.
func Migrate(db *sql.DB, directory string, version int) error {
	return WithDirectory(directory).WithRevision(version).Apply(db)
}
