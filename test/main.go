package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	local "github.com/sbowman/migrations/cmd"
	remote "github.com/sbowman/migrations/remote/cmd"

	_ "github.com/lib/pq"
)

// URI of the PostgreSQL database for testing
const URI = "postgres://postgres@localhost/migrations?sslmode=disable"

func main() {
	viper.Set("uri", URI)

	root := &cobra.Command{
		Use:   "migrations",
		Short: "Migrations testing app",
	}

	root.PersistentFlags().Bool("remote", false, "enable remote S3 migrations for testing")

	testRemote := os.Getenv("TEST_REMOTE") == "true"
	if testRemote {
		remote.AddRemoteTo(root)
	} else {
		local.AddTo(root)
	}

	// Don't run 3 yet
	viper.Set("revision", 2)
	if err := root.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to run: %s\n", err.Error())
	}

	// viper.Set("revision", 3)
	// if err := root.Execute(); err == nil {
	// 	_, _ = fmt.Fprintf(os.Stderr, "Expected revision 3 to error out; it did not")
	// 	os.Exit(1)
	// }
	//
	// conn, err := sql.Open("postgres", URI)
	// if err != nil {
	// 	_, _ = fmt.Fprintf(os.Stderr, "Unable to connect to the database: %s", err)
	// 	os.Exit(1)
	// }
	//
	// // Revision 3 should not be in the schema migrations table
	// row := conn.QueryRow("select migration from schema_migrations where migration = $1 limit 1 for update", "3-no-tx.sql")
	// if row.Scan() != sql.ErrNoRows {
	// 	_, _ = fmt.Fprintln(os.Stderr, "Didn't expect 3-no-tx.sql to complete successfully")
	// 	os.Exit(1)
	// }
	//
	// // Check to see if "abc" is there but not "zzz"
	// rows, err := conn.Query("select name from sample")
	// if err == sql.ErrNoRows {
	// 	_, _ = fmt.Fprintln(os.Stderr, "No data in the sample table; expected the abc row")
	// 	os.Exit(1)
	// } else if err != nil {
	// 	_, _ = fmt.Fprintf(os.Stderr, "Couldn't query the sample table: %s\n", err)
	// 	os.Exit(1)
	// }
	//
	// for rows.Next() {
	// 	var name string
	// 	if err := rows.Scan(&name); err != nil {
	// 		if name != "abc" {
	// 			_, _ = fmt.Fprintf(os.Stderr, "Unexpected entry in sample; got %s\n", name)
	// 		}
	// 	}
	// }
	//
	// // Put migration 3 into the schemas table, then rollback to 2, to make sure "down" works
	// // as well
	// _, err = conn.Exec("insert into schema_migrations (migration) values ($1)", "3-no-tx-sql")
	// if err != nil {
	// 	_, _ = fmt.Fprintf(os.Stderr, "Failed to insert migration: %s\n", err)
	// 	os.Exit(1)
	// }
	//
	// viper.Set("revision", 2)
	// if err := root.Execute(); err == nil {
	// 	_, _ = fmt.Fprintf(os.Stderr, "Expected revision 3 to error out rolling back; it did not")
	// 	os.Exit(1)
	// }
	//
	// rows, err = conn.Query("select name from sample")
	// if err != sql.ErrNoRows {
	// 	_, _ = fmt.Fprintln(os.Stderr, "Found rows in the sample table; the migration should have removed them")
	// 	os.Exit(1)
	// }
}
