package tests_test

import (
	"os"
	"testing"

	v1 "github.com/sbowman/migrations"
	v2 "github.com/sbowman/migrations/v2"
)

// Can we upgrade from migrations v1 to v2?
func TestUpgrade(t *testing.T) {
	defer clean(t)

	// V1 Migration
	if err := v1.Migrate(conn, "./sql_upgrade", 2); err != nil {
		t.Fatalf("Unable to run v1 migrations: %s", err)
	}

	if err := tableExists("schema_migrations"); err != nil {
		t.Fatal("The schema_migrations table wasn't created")
	}

	if err := tableExists("users"); err != nil {
		t.Fatal("Users table not found in database")
	}

	if err := tableExists("roles"); err != nil {
		t.Fatal("Roles table not found in database")
	}

	if _, err := conn.Exec("insert into users (email, username) values ('jdoe@nowhere.com', 'jdoe')"); err != nil {
		t.Errorf("Unable to insert record into users: %s", err)
	}

	rows, err := conn.Query("select email from users where username = 'jdoe'")
	if err != nil {
		t.Errorf("Didn't find expected record in database: %s", err)
	}

	var email string
	for rows.Next() {
		if err := rows.Scan(&email); err != nil {
			t.Errorf("Failed to get email from database: %s", err)
		}

		if email != "jdoe@nowhere.com" {
			t.Errorf("Expected name jdoe@nowhere.com, got %s", email)
		}
	}

	if email == "" {
		t.Error("Email not found")
	}

	// Upgrade to V2
	if err := v2.InitializeDB(conn, ".sql_upgrade"); err != nil {
		t.Fatalf("Failed upgrade database: %s", err)
	}

	// Did it work?
	if err := tableExists("migrations.applied"); err != nil {
		t.Error("Migrations applied table not found in database")
	}

	if err := tableExists("migrations.rollbacks"); err != nil {
		t.Error("Migrations applied table not found in database")
	}

	if err := tableExists("schema_migrations"); err == nil {
		t.Error("The schema_migrations table was found in database; should have been removed")
	}

	if err := migrationApplied("1-create-users.sql"); err != nil {
		t.Error("Did not migration 1-create-users.sql")
	}

	if err := migrationApplied("2-create-roles.sql"); err != nil {
		t.Error("Did not migration 1-create-roles.sql")
	}

	if err := migrationApplied("3-alter-users.sql"); err == nil {
		t.Error("Migration 3-alter-users.sql was prematurely applied")
	}

	if err := v2.WithDirectory("./sql_upgrade").Apply(conn); err != nil {
		t.Errorf("Failed to run v2 migrations: %s", err)
	}

	if err := migrationApplied("3-alter-users.sql"); err != nil {
		t.Error("Did not migrate 3-alter-users.sql")
	}

	err = os.Rename("./sql_upgrade/3-alter-users.sql", "./sql_upgrade/skip_3-alter-users.sql")
	defer func() {
		_ = os.Rename("./sql_upgrade/skip_3-alter-users.sql", "./sql_upgrade/3-alter-users.sql")
	}()

	if err != nil {
		t.Fatalf("Could not move third migration out of the way: %s", err)
	}

	// See if a rollback works
	if err := v2.WithDirectory("./sql_upgrade").Apply(conn); err != nil {
		t.Errorf("Failed to run v2 migrations: %s", err)
	}

	if err := migrationApplied("1-create-users.sql"); err != nil {
		t.Error("Did not migration 1-create-users.sql")
	}

	if err := migrationApplied("2-create-roles.sql"); err != nil {
		t.Error("Did not migration 1-create-roles.sql")
	}

	if err := migrationApplied("3-alter-users.sql"); err == nil {
		t.Error("Migration 3-alter-users.sql remains")
	}
}

// Can we downgrade from migrations v2 to v1?
func TestDowngrade(t *testing.T) {
	defer clean(t)

	// V1 Migration
	if err := v1.Migrate(conn, "./sql_upgrade", 2); err != nil {
		t.Fatalf("Unable to run v1 migrations: %s", err)
	}

	if err := tableExists("schema_migrations"); err != nil {
		t.Fatal("The schema_migrations table wasn't created")
	}

	if err := tableExists("users"); err != nil {
		t.Fatal("Users table not found in database")
	}

	if err := tableExists("roles"); err != nil {
		t.Fatal("Roles table not found in database")
	}

	if _, err := conn.Exec("insert into users (email, username) values ('jdoe@nowhere.com', 'jdoe')"); err != nil {
		t.Errorf("Unable to insert record into users: %s", err)
	}

	rows, err := conn.Query("select email from users where username = 'jdoe'")
	if err != nil {
		t.Errorf("Didn't find expected record in database: %s", err)
	}

	var email string
	for rows.Next() {
		if err := rows.Scan(&email); err != nil {
			t.Errorf("Failed to get email from database: %s", err)
		}

		if email != "jdoe@nowhere.com" {
			t.Errorf("Expected name jdoe@nowhere.com, got %s", email)
		}
	}

	if email == "" {
		t.Error("Email not found")
	}

	// Upgrade to V2
	if err := v2.InitializeDB(conn, ".sql_upgrade"); err != nil {
		t.Fatalf("Failed upgrade database: %s", err)
	}

	// Did it work?
	if err := tableExists("migrations.applied"); err != nil {
		t.Error("Migrations applied table not found in database")
	}

	if err := tableExists("migrations.rollbacks"); err != nil {
		t.Error("Migrations applied table not found in database")
	}

	if err := tableExists("schema_migrations"); err == nil {
		t.Error("The schema_migrations table was found in database; should have been removed")
	}

	if err := migrationApplied("1-create-users.sql"); err != nil {
		t.Error("Did not migration 1-create-users.sql")
	}

	if err := migrationApplied("2-create-roles.sql"); err != nil {
		t.Error("Did not migration 1-create-roles.sql")
	}

	if err := migrationApplied("3-alter-users.sql"); err == nil {
		t.Error("Migration 3-alter-users.sql was prematurely applied")
	}

	if err := v2.WithDirectory("./sql_upgrade").Apply(conn); err != nil {
		t.Errorf("Failed to run v2 migrations: %s", err)
	}

	if err := migrationApplied("3-alter-users.sql"); err != nil {
		t.Error("Did not migrate 3-alter-users.sql")
	}

	// Downgrade back to v1
	err = v2.Downgrade(conn)
	if err != nil {
		t.Error(err.Error())
	}

	if err := tableExists("migrations.applied"); err == nil {
		t.Error("Migrations applied table not found in database")
	}

	if err := tableExists("migrations.rollbacks"); err == nil {
		t.Error("Migrations applied table not found in database")
	}

	if err := tableExists("schema_migrations"); err != nil {
		t.Fatal("The schema_migrations table wasn't created")
	}

	if err := tableExists("users"); err != nil {
		t.Fatal("Users table not found in database")
	}

	if err := tableExists("roles"); err != nil {
		t.Fatal("Roles table not found in database")
	}

	if _, err := conn.Exec("insert into users (email, username, age) values ('bob@nowhere.com', 'bob', 99)"); err != nil {
		t.Errorf("Unable to insert record into users: %s", err)
	}

	rows, err = conn.Query("select email from users where username = 'bob'")
	if err != nil {
		t.Errorf("Didn't find expected record in database: %s", err)
	}

	for rows.Next() {
		if err := rows.Scan(&email); err != nil {
			t.Errorf("Failed to get email from database: %s", err)
		}

		if email != "bob@nowhere.com" {
			t.Errorf("Expected name bob@nowhere.com, got %s", email)
		}
	}

	if email == "" {
		t.Error("Email not found")
	}

	// See if a v1 rollback works
	if err := v1.Rollback(conn, "./sql_upgrade", 2); err != nil {
		t.Fatalf("Unable to rollback migration to revision 1: %s", err)
	}

	_, err = conn.Query("select name from roles where name = 'Empty'")
	if err == nil {
		t.Error("Expected querying for the rolled-back table to fail")
	}

	_, err = conn.Query("select age from users where name = 'Bob'")
	if err == nil {
		t.Error("Expected querying for the rolled-back column to fail")
	}

}
