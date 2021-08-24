package tests_test

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/sbowman/migrations"

	_ "github.com/jackc/pgx/v4/stdlib"
)

const (
	// TableExists queries for the table in the PostgreSQL metadata.
	TableExists = `
select exists 
    (select from information_schema.tables 
            where table_schema = 'public' and 
                  table_name = $1)`
)

var conn *sql.DB

func TestMain(m *testing.M) {
	var err error

	migrations.Log = new(migrations.NilLogger)

	conn, err = sql.Open("pgx", "postgres://postgres@localhost/migrations_test?sslmode=disable")
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Unable to connect to migrations_test database: %s\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

// TestMatch checks that we can match "up" and "down" sections in the files.
func TestMatch(t *testing.T) {
	doc, _, err := migrations.ReadSQL("./sql/match_hash.txt", migrations.Up)
	if err != nil {
		t.Errorf("Unable to parsh hashed up: %s", err)
	}

	if strings.TrimSpace(string(doc)) != "Matched Up" {
		t.Errorf(`Expected "Matched Up", but got "%s"`, strings.TrimSpace(string(doc)))
	}

	doc, _, err = migrations.ReadSQL("./sql/match_hash.txt", migrations.Down)
	if err != nil {
		t.Errorf("Unable to parsh hashed down: %s", err)
	}

	if strings.TrimSpace(string(doc)) != "Matched Down" {
		t.Errorf(`Expected "Matched Down", but got "%s"`, strings.TrimSpace(string(doc)))
	}

	doc, _, err = migrations.ReadSQL("./sql/match_no_hash.txt", migrations.Up)
	if err != nil {
		t.Errorf("Unable to parsh no hashed up: %s", err)
	}

	if strings.TrimSpace(string(doc)) != "Matched Up" {
		t.Errorf(`Expected "Matched Up", but got "%s"`, strings.TrimSpace(string(doc)))
	}

	doc, _, err = migrations.ReadSQL("./sql/match_no_hash.txt", migrations.Down)
	if err != nil {
		t.Errorf("Unable to parsh no hashed down: %s", err)
	}

	if strings.TrimSpace(string(doc)) != "Matched Down" {
		t.Errorf(`Expected "Matched Down", but got "%s"`, strings.TrimSpace(string(doc)))
	}
}

// TestUp confirms upward bound migrations work.
func TestUp(t *testing.T) {
	defer clean(t)

	if err := migrate(1); err != nil {
		t.Fatalf("Unable to run migration: %s", err)
	}

	if err := tableExists("schema_migrations"); err != nil {
		t.Fatal("The schema_migrations table wasn't created")
	}

	if err := tableExists("samples"); err != nil {
		t.Fatal("Sample table not found in database")
	}

	if _, err := conn.Exec("insert into samples (name) values ('Bob')"); err != nil {
		t.Errorf("Unable to insert record into samples: %s", err)
	}

	rows, err := conn.Query("select name from samples where name = 'Bob'")
	if err != nil {
		t.Errorf("Didn't find expected record in database: %s", err)
	}

	var name string
	for rows.Next() {
		if err := rows.Scan(&name); err != nil {
			t.Errorf("Failed to get name from database: %s", err)
		}

		if name != "Bob" {
			t.Errorf("Expected name Bob, got %s", name)
		}
	}

	if name == "" {
		t.Error("Name not found")
	}
}

// Make sure revisions, i.e. partial migrations, are working.
func TestRevisions(t *testing.T) {
	defer clean(t)

	if err := migrate(1); err != nil {
		t.Fatalf("Unable to run migration to revision 1: %s", err)
	}

	if _, err := conn.Exec("insert into samples (name, email) values ('Bob', 'bob@home.com')"); err == nil {
		t.Error("Expected inserting an email address to fail")
	}

	if err := migrate(2); err != nil {
		t.Fatalf("Unable to run migration to revision 2: %s", err)
	}

	if _, err := conn.Exec("insert into samples (name, email) values ('Bob', 'bob@home.com')"); err != nil {
		t.Errorf("Expected to be able to insert email address after revision 2: %s", err)
	}

	rows, err := conn.Query("select email from samples where name = 'Bob'")
	if err != nil {
		t.Errorf("Didn't find expected record in database: %s", err)
	}

	var email string
	for rows.Next() {
		if err := rows.Scan(&email); err != nil {
			t.Errorf("Failed to get email from database: %s", err)
		}

		if email != "bob@home.com" {
			t.Errorf("Expected email bob@home.com for Bob, got %s", email)
		}
	}

	if email == "" {
		t.Error("Email not found")
	}
}

// Make sure migrations can be rolled back.
func TestDown(t *testing.T) {
	defer clean(t)

	if err := migrate(2); err != nil {
		t.Fatalf("Unable to run migration to revision 2: %s", err)
	}

	if _, err := conn.Exec("insert into samples (name, email) values ('Bob', 'bob@home.com')"); err != nil {
		t.Errorf("Expected to be able to insert email address after revision 2: %s", err)
	}

	rows, err := conn.Query("select email from samples where name = 'Bob'")
	if err != nil {
		t.Errorf("Didn't find expected record in database: %s", err)
	}

	var email string
	for rows.Next() {
		if err := rows.Scan(&email); err != nil {
			t.Errorf("Failed to get email from database: %s", err)
		}

		if email != "bob@home.com" {
			t.Errorf("Expected email bob@home.com for Bob, got %s", email)
		}
	}

	if email == "" {
		t.Error("Email not found")
	}

	// Rollback
	if err := migrate(1); err != nil {
		t.Fatalf("Unable to run migration to revision 1: %s", err)
	}

	if _, err := conn.Exec("insert into samples (name, email) values ('Alice', 'alice@home.com')"); err == nil {
		t.Error("Expected inserting an email address to fail")
	}

	_, err = conn.Query("select email from samples where name = 'Bob'")
	if err == nil {
		t.Error("Expected an error, as the email column shouldn't exist")
	}

	rows, err = conn.Query("select name from samples where name = 'Alice'")
	if err != nil {
		t.Errorf("Unable to query for samples: %s", err)
	}

	for rows.Next() {
		t.Errorf("Did not expect results from the query")
	}
}

// Is the simplified Rollback function working?
func TestRollback(t *testing.T) {
	defer clean(t)

	if err := migrate(2); err != nil {
		t.Fatalf("Unable to run migration to revision 2: %s", err)
	}

	if _, err := conn.Exec("insert into samples (name, email) values ('Bob', 'bob@home.com')"); err != nil {
		t.Errorf("Expected insert to succeed: %s", err)
	}

	if err := migrations.Rollback(conn, "./sql", 1); err != nil {
		t.Fatalf("Unable to rollback migration to revision 1: %s", err)
	}

	_, err := conn.Query("select email from samples where name = 'Bob'")
	if err == nil {
		t.Error("Expected querying for the rolled-back column to fail")
	}
}

// Under normal circumstances, if part of a migration fails, the whole migration false.
func TestTransactions(t *testing.T) {
	defer clean(t)

	if err := migrate(3); err == nil {
		t.Error("Expected migration to fail")
	}

	rows, err := conn.Query("select name from samples where name = 'abc'")
	if err != nil {
		t.Fatalf("Unable to query for sample names:%s", err)
	}

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Errorf("Unable to scan results: %s", err)
			continue
		}

		if name == "abc" {
			t.Error("Unexpected abc value")
		}
	}
}

// Does the /async flag run the migration commands asynchronously?
func TestAsyncFlag(t *testing.T) {
	defer clean(t)

	if err := migrate(2); err != nil {
		t.Fatalf("Unable to run migration to revision 2: %s", err)
	}

	skip(t, "3-check-tx.sql")

	// Need to wait for the results to check success
	asyncResults, err := migrations.MigrateAsync(conn, "./sql", 4)
	if err != nil {
		t.Fatalf("Migrations failed: %s", err)
	}

	for result := range asyncResults {
		if result.Err != nil {
			t.Fatalf("Asychronous migration failed: %s", err)
		}

		if result.Migration != "./sql/4-check-async.sql" {
			t.Errorf(`Expected result to include migration "./sql/4-check-async.sql," but got %s`, result.Migration)
		}
		expected := []string{
			"aaa",
			"ccc",
		}

		for _, check := range expected {
			rows, err := conn.Query("select name from samples where name = $1", check)
			if err != nil {
				t.Errorf("Unable to query for name %s: %s", check, err)
			}

			var name string
			for rows.Next() {
				if err := rows.Scan(&name); err != nil {
					t.Error("Unable to scan result")
				}

				if name == check {
					break
				}
			}

			if name == "" {
				t.Errorf("Expected a %s record; didn't find one", check)
			}
		}
	}

	// Make sure the migrations succeeded
	rows, err := conn.Query("select migration from schema_migrations")
	if err != nil {
		t.Errorf("Unable to query for migrations: %s", err)
	}

	var count int
	var found bool
	for rows.Next() {
		var migration string
		if err := rows.Scan(&migration); err != nil {
			t.Errorf("Unable to get migration data: %s", err)
			continue
		}

		count++

		if migration == "4-check-async.sql" {
			found = true
		}
	}

	if count != 4 {
		t.Errorf("Expected four migrations; found %d", count)
	}

	if !found {
		t.Errorf("The async migration was not logged in the schema_migrations table")
	}
}

// Make sure migrations "complete" before long-running asynchronous migrations complete.
//
// THIS TEST TAKES 3 SECONDS TO COMPLETE!
func TestSlowAsync(t *testing.T) {
	defer clean(t)

	if err := migrate(2); err != nil {
		t.Fatalf("Unable to run migration to revision 2: %s", err)
	}

	skip(t, "3-check-tx.sql")

	// Migration 5 will take five seconds to run
	asyncResults, err := migrations.MigrateAsync(conn, "./sql", 5)
	if err != nil {
		t.Fatalf("Migrations failed: %s", err)
	}

	// Migration 5 should be marked as completed before it's done
	rows, err := conn.Query("select migration from schema_migrations where migration = '5-slow-async.sql'")
	if err != nil {
		t.Errorf("Unable to query for migrations: %s", err)
	}

	var migration string
	for rows.Next() {
		if err := rows.Scan(&migration); err != nil {
			t.Errorf("Unable to get migration data: %s", err)
			continue
		}
	}

	if migration == "" {
		t.Error("Expected the schema_migration record to exist for the slow async query migration")
	}

	rows, err = conn.Query("select name from samples where name = 'slowup'")
	if err != nil {
		t.Errorf("Unable to query samples: %s", err)
	}

	var missing string
	for rows.Next() {
		if err := rows.Scan(&missing); err != nil {
			t.Errorf("Unable to get samples data: %s", err)
			continue
		}
	}

	if missing != "" {
		t.Errorf("Expected test record after slow async query to not be there yet, but it was: %s", missing)
	}

	// Wait for the slow request to complete, and the value after the slow query should be there
	for result := range asyncResults {
		if result.Err != nil {
			t.Fatalf("Asychronous migration failed: %s", err)
		}

		if result.Migration != "./sql/5-slow-async.sql" {
			continue
		}

		rows, err = conn.Query("select name from samples where name = 'slowup'")
		if err != nil {
			t.Errorf("Unable to query samples data: %s", err)
		}

		var found bool
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				t.Errorf("Unable to scan result: %s", err)
			}

			if name == "slowup" {
				found = true
				break
			}
		}

		if !found {
			t.Error("Expected a 'slowup' record; didn't find one")
		}
	}
}

// Test that the migration gets recorded even when an async migration failed.
//
// THIS TEST TAKES 3 SECONDS TO COMPLETE!
func TestAsyncFailure(t *testing.T) {
	defer clean(t)

	if err := migrate(2); err != nil {
		t.Fatalf("Unable to run migration to revision 2: %s", err)
	}

	skip(t, "3-check-tx.sql")
	skip(t, "5-slow-async.sql")

	// Migration 6 will fail
	asyncResults, err := migrations.MigrateAsync(conn, "./sql", 6)
	if err != nil {
		t.Fatalf("Migrations failed: %s", err)
	}

	// Migration 6 should be marked as completed, even though it fails
	rows, err := conn.Query("select migration from schema_migrations where migration = '6-bad-async.sql'")
	if err != nil {
		t.Errorf("Unable to query for migrations: %s", err)
	}

	var migration string
	for rows.Next() {
		if err := rows.Scan(&migration); err != nil {
			t.Errorf("Unable to get migration data: %s", err)
			continue
		}
	}

	if migration == "" {
		t.Error("Expected the schema_migration record to exist for the bad async query migration")
	}

	rows, err = conn.Query("select name from samples where name = 'slowup'")
	if err != nil {
		t.Errorf("Unable to query samples: %s", err)
	}

	for result := range asyncResults {
		if result.Migration != "./sql/6-bad-async.sql" {
			continue
		}

		if result.Err == nil {
			t.Error("Expected 6-bad-async.sql to fail")
		}

		if result.Migration != "./sql/6-bad-async.sql" {
			t.Errorf(`Expected "./sql/6-bad-async.sql" in the migration; was "%s"`, result.Migration)
		}

		if result.Command != "insert into samples (blah) values ('noway')" {
			t.Errorf(`Expected "insert into samples (blah) values ('noway')" in the command; was "%s"`, result.Command)
		}
	}
}

// Shortcut to run the test migrations in the sql directory.
func migrate(revision int) error {
	if os.Getenv("NOTX") != "true" {
		return migrations.Migrate(conn, "./sql", revision)
	}

	return migrations.MigrateUnsafe(conn, "./sql", revision)
}

// Clean out the database.
func clean(t *testing.T) {
	if _, err := conn.Exec("delete from schema_migrations"); err != nil {
		t.Fatalf("Unable to clear the schema_migrations table: %s", err)
	}

	rows, err := conn.Query("select table_name from information_schema.tables where table_schema='public'")
	if err != nil {
		t.Fatalf("Couldn't query for table names: %s", err)
	}

	var name string
	for rows.Next() {
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("Failed to get table name: %s", err)
		}

		// Note: not exactly safe, but this is just a test case
		if _, err := conn.Exec("drop table if exists " + name); err != nil {
			t.Fatalf("Couldn't drop table %s: %s", name, err)
		}
	}

	if name == "" {
		t.Error("Name not found")
	}
}

// Check if the table exists.  Returns nil if the table exists.
func tableExists(table string) error {
	rows, err := conn.Query(TableExists, table)
	if err != nil {
		return err
	}

	if rows.Next() {
		var found bool
		if err := rows.Scan(&found); err != nil {
			return err
		}

		if found {
			return nil
		}
	}

	return sql.ErrNoRows
}

// Skip a migration by adding a record to schema_migrations.
func skip(t *testing.T, path string) {
	if _, err := conn.Exec("insert into schema_migrations values ($1)", path); err != nil {
		t.Fatalf("Failed to skip %s migration: %s", path, err)
	}
}
