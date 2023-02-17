package migrations

import "database/sql"

// UpgradeMigrations upgrades your application from migrations/v1 to migrations/v2.  Basically
// copies the migrations from the schema_migrations table to the migrations.applied table.
func UpgradeMigrations(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	if err := CreateMigrationsSchema(tx); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := CreateMigrationsApplied(tx); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := CreateMigrationsRollbacks(tx); err != nil {
		_ = tx.Rollback()
		return err
	}

	if MissingSchemaMigrations(tx) {
		return tx.Commit()
	}

	if _, err := tx.Exec("insert into migrations.applied(migration) " +
		"select migration from schema_migrations on conflict migration do nothing"); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := dropSchemaMigrations(tx); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}

// DowngradeMigrations rolls your database back from migrations/v2 to a migrations/v1-compatible
// database, or specifically, recreate schema_migrations and copy migrations.applied into the
// schema_migrations table.
func DowngradeMigrations(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	if err := CreateSchemaMigrations(tx); err != nil {
		_ = tx.Rollback()
		return err
	}

	if MissingMigrationsApplied(tx) {
		return tx.Commit()
	}

	if _, err := tx.Exec("insert into schema_migrations(migration) " +
		"select migration from migrations.applied on conflict migration do nothing"); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := dropMigrationsSchema(tx); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}

// CreateSchemaMigrations creates the schema_migrations table in the database
// if it doesn't already exist.
func CreateSchemaMigrations(tx *sql.Tx) error {
	if MissingSchemaMigrations(tx) {
		Log.Infof("Creating schema_migrations table in the database")
		if _, err := tx.Exec("create table schema_migrations(migration varchar(1024) not null primary key)"); err != nil {
			return err
		}
	}

	return nil
}

// MissingSchemaMigrations returns true if there is no schema_migrations table
// in the database.
func MissingSchemaMigrations(tx *sql.Tx) bool {
	row := tx.QueryRow("select not(exists(select 1 from pg_catalog.pg_class c " +
		"join pg_catalog.pg_namespace n " +
		"on n.oid = c.relnamespace " +
		"where n.nspname = 'public' and c.relname = 'schema_migrations'))")

	var result bool
	if err := row.Scan(&result); err != nil {
		return true
	}

	return result
}

// dropSchemaMigrations deletes the migrations/v1 table.  Should only be called from
// UpgradeMigrations.
func dropSchemaMigrations(tx *sql.Tx) error {
	if _, err := tx.Exec("drop table schema_migrations"); err != nil {
		return err
	}

	return nil
}

// dropMigrationsSchema deletes the migrations/v2 tables.  Should only be called from
// DowngradeMigrations.
func dropMigrationsSchema(tx *sql.Tx) error {
	if _, err := tx.Exec("drop table migrations.rollbacks"); err != nil {
		return err
	}

	if _, err := tx.Exec("drop table migrations.applied"); err != nil {
		return err
	}

	if _, err := tx.Exec("drop schema migrations"); err != nil {
		return err
	}

	return nil
}
