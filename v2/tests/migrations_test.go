package tests_test

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/sbowman/migrations/v2"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const (
	// TableExists queries for the table in the PostgreSQL metadata.
	TableExists = `
select exists 
    (select from information_schema.tables 
            where table_schema = $1 and 
                  table_name = $2)`
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

	if err := tableExists("migrations.applied"); err != nil {
		t.Fatal("The migrations.applied table wasn't created")
	}

	if err := tableExists("migrations.rollbacks"); err != nil {
		t.Fatal("The migrations.rollbacks wasn't created")
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

	// Check that rollbacks are loaded in the database
	rows, err = conn.Query("select migration, down from migrations.rollbacks")
	if err != nil {
		t.Fatalf("Failed to query for rollback DDL: %s", err)
	}

	var found int

	var migration, down string
	for rows.Next() {
		found++

		if err := rows.Scan(&migration, &down); err != nil {
			t.Fatalf("Failed to get applied migration from the database: %s", err)
		}

		SQL, _, err := migrations.ReadSQL(migration, migrations.Down)
		if err != nil {
			t.Fatalf("Unable to read SQL file %s: %s", migration, err)
		}

		if SQL != migrations.SQL(down) {
			t.Errorf("Expected down migration %s to equal %s", SQL, migration)
		}
	}

	if found == 0 {
		t.Error("Didn't find any rollbacks")
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

// Shortcut to run the test migrations in the sql directory.
func migrate(revision int) error {
	return migrations.WithRevision(revision).Apply(conn)
}

// Clean out the database.
func clean(t *testing.T) {
	if _, err := conn.Exec("delete from migrations.applied"); err != nil {
		t.Fatalf("Unable to clear the migrations.applied table: %s", err)
	}

	if _, err := conn.Exec("delete from migrations.rollbacks"); err != nil {
		t.Fatalf("Unable to clear the migrations.rollbacks table: %s", err)
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
	parts := strings.Split(table, ".")

	var schema string
	if len(parts) == 1 {
		schema = "public"
		table = parts[0]
	} else {
		schema = parts[0]
		table = parts[1]
	}

	rows, err := conn.Query(TableExists, schema, table)
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
