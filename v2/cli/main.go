package main

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/sbowman/migrations/v2"
	"github.com/spf13/cobra"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const (
	// URI is the database URI used to connect to the database to run migrations.
	URI = "uri"

	// Migrations is the name of the db.migrations spf13/viper setting.
	Migrations = "migrations"

	// Revision is the database migration revision setting name.
	Revision = "revision"
)

func createCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create",
		Short: "Create new database migrations from the template",

		Run: func(cmd *cobra.Command, args []string) {
			directory, _ := cmd.PersistentFlags().GetString(Migrations)

			for _, arg := range args {
				if err := migrations.Create(directory, arg); err != nil {
					migrations.Log.Infof(err.Error())
					os.Exit(1)
				}
			}
		},
	}
}

func runCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run the database schema migrations",

		Run: func(cmd *cobra.Command, args []string) {
			uri, err := cmd.PersistentFlags().GetString(URI)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "A --%s setting is required\n", URI)
				return
			}

			directory, _ := cmd.PersistentFlags().GetString(Migrations)

			revision, err := cmd.Flags().GetInt(Revision)
			if err != nil {
				revision = migrations.Latest
			}

			if revision >= 0 {
				migrations.Log.Infof("Migrating %s to revision %d", uri, revision)
			} else {
				migrations.Log.Infof("Migrating %s to the latest revision", uri)
			}

			if err := runMigrations(uri, directory, revision); err != nil {
				migrations.Log.Infof("Failed to migrate: %s", err)
			}
		},
	}

	cmd.Flags().Int(Revision, migrations.Latest, "migrate to this revision; defaults to latest")

	return cmd
}

func runMigrations(uri, directory string, revision int) error {
	conn, err := sql.Open("pgx", uri)
	if err != nil {
		return err
	}

	migrations.Log.Infof("Running migrations in %s...", directory)
	if err := migrations.Migrate(conn, directory, revision); err != nil {
		migrations.Log.Infof(err.Error())
		os.Exit(1)
	}

	return nil
}

func main() {
	cmd := &cobra.Command{
		Use:   "migrations",
		Short: "CLI for running database migrations",
	}

	cmd.PersistentFlags().String(Migrations, "./sql", "path to database migration (*.sql) files")

	cmd.AddCommand(createCmd())
	cmd.AddCommand(runCmd())

	if err := cmd.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, err.Error())
		os.Exit(1)
	}
}
