package migrations

import "os"

// EnvMigrations is the environment variable that can be used to point to the directory of SQL
// migrations.
const EnvMigrations = "MIGRATIONS"

type Options struct {
	// Revision is the revision to forcibly move to.  Defaults to the latest revision as
	// indicated by the available SQL files (which could be a rollback if the applied
	// migrations exceed the latest SQL file.
	Revision int

	// Directory is the directory containing the SQL files.  Defaults to the "./sql" directory.
	Directory string
}

// DefaultOptions returns the defaults for the migrations package.  Revision defaults to the
// latest revision, and the directory defaults to what's defined in the MIGRATIONS environment
// variable, or "./sql" if the environment variable was not defined..
func DefaultOptions() Options {
	directory := os.Getenv(EnvMigrations)
	if directory == "" {
		directory = "./sql"
	}

	return Options{
		Revision:  Latest,
		Directory: directory,
	}
}

// WithRevision manually indicates the revision to migrate the database to.  By default, the
// migrations to get the database to the revision indicated by the latest SQL migraiton file is
// used.
func WithRevision(revision int) Options {
	return DefaultOptions().WithRevision(revision)
}

// WithDirectory points to the directory of SQL migrations files that should be used to migrate
// the database schema.  Defaults to the "./sql" directory.
func WithDirectory(path string) Options {
	return DefaultOptions().WithDirectory(path)
}

// WithRevision manually indicates the revision to migrate the database to.  By default, the
// migrations to get the database to the revision indicated by the latest SQL migraiton file is
// used.
func (options Options) WithRevision(revision int) Options {
	options.Revision = revision
	return options
}

// WithDirectory points to the directory of SQL migrations files that should be used to migrate
// the database schema.  Defaults to the "./sql" directory.
func (options Options) WithDirectory(path string) Options {
	options.Directory = path
	return options
}
