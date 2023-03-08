package tests_test

import "testing"

func TestEmbeddedRollback(t *testing.T) {
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

// Test the /stop flag
func TestEmbeddedRollbackStop(t *testing.T) {
	// TODO
}
