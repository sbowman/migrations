package tests_test

import (
	"fmt"
	"testing"

	"github.com/sbowman/migrations/v2"
)

func TestEmbeddedRollback(t *testing.T) {
	defer clean(t)

	// TODO: need to test a SQL only rollback!

	if err := migrations.WithDirectory("./sql_embedded").WithRevision(migrations.Latest).Apply(conn); err != nil {
		t.Fatalf("Unable to run migration to latest revision: %s", err)
	}

	// Add another migration that doesn't exist as a file
	revision := migrations.LatestRevision(migrations.DefaultOptions().Directory)

	tx, err := conn.Begin()
	if err != nil {
		t.Fatalf("Can't create a transaction! %s", err)
	}

	migration := fmt.Sprintf("%d-create-user-roles.sql", revision)

	if _, err := tx.Exec("insert into migrations.applied values ($1)", migration); err != nil {
		t.Fatalf("Can't insert extra migration: %s", err)
	}

	if _, err := tx.Exec("insert into migrations.rollbacks (migration, down) values ($1, $2)", migration, "drop table user_roles;"); err != nil {
		t.Fatalf("Can't insert extra rollback: %s", err)
	}

	if _, err := tx.Exec("create table user_roles (user_id integer not null references users (id), role_id integer not null references roles (id), primary key (user_id, role_id))"); err != nil {
		t.Fatalf("Can't create table for rollback: %s", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("Failed to commit changes: %s", err)
	}

	if err := tableExists("user_roles"); err != nil {
		t.Fatalf("Failed to create user_roles table: %s", err)
	}

	// Migrating to "latest" should take things back one level, because our above migration
	// doesn't exist as a SQL file
	if err := migrations.WithDirectory("./sql_embedded").WithRevision(migrations.Latest).Apply(conn); err != nil {
		t.Fatalf("Unable to run migration to latest revision: %s", err)
	}

	if err := tableExists("user_roles"); err == nil {
		t.Errorf("Expected user_roles table to be gone")
	}

	// Is the rollback in the database gone?
	row := conn.QueryRow("select not exists(select migration from migrations.rollbacks where migration = $1)", migration)
	if row == nil {
		t.Errorf("Unable to query for rollback: %s", err)
	} else {
		var found bool
		if err := row.Scan(&found); err != nil {
			t.Errorf("Unable to query for rollback: %s", err)
		} else if found {
			t.Errorf("Failed to delete the rollback migration for %s", migration)
		}
	}

	// Is the migration in the database gone?
	row = conn.QueryRow("select not exists(select migration from migrations.applied where migration = $1)", migration)
	if row == nil {
		t.Errorf("Unable to query for applied migration: %s", err)
	} else {
		var found bool
		if err := row.Scan(&found); err != nil {
			t.Errorf("Unable to query for applied migration: %s", err)
		} else if found {
			t.Errorf("Failed to delete the applied migration for %s", migration)
		}
	}
}

// Test the /stop flag
func TestEmbeddedRollbackStop(t *testing.T) {
	// TODO
}
