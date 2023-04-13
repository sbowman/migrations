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
	// CurrentVersion is the version of this migrations library.
	CurrentVersion = "2.0.0"

	// URI is the database URI used to connect to the database to run migrations.
	URI = "uri"

	// Migrations is the name of the db.migrations spf13/viper setting.
	Migrations = "migrations"

	// Revision is the database migration revision setting name.
	Revision = "revision"

	// DisableEmbeddedRollbacks is a flag that disables the embedded rollbacks functionality.
	DisableEmbeddedRollbacks = "no-rollback"

	// Version is the version of the migration library used by the command-line application.
	Version = "version"
)

func createCmd() *cobra.Command {
	cmd := &cobra.Command{
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

	cmd.Flags().String(Migrations, "./sql", "path to database migration (*.sql) files")

	return cmd
}

func applyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply the database schema migrations",

		Run: func(cmd *cobra.Command, args []string) {
			uri, err := getURI(cmd)
			if err != nil {
				migrations.Log.Infof("A --%s setting or DB_URI or DB environment variable is required!", URI)
				os.Exit(1)
			}

			directory, _ := cmd.Flags().GetString(Migrations)
			revision, _ := cmd.Flags().GetInt(Revision)

			if revision >= 0 {
				migrations.Log.Infof("Migrating %s to revision %d", uri, revision)
			} else {
				migrations.Log.Infof("Migrating %s to the latest revision", uri)
			}

			conn, err := sql.Open("pgx", uri)
			if err != nil {
				migrations.Log.Infof("Failed to connect to database %s: %s", uri, err)
				os.Exit(1)
			}

			options := migrations.WithDirectory(directory).WithRevision(revision)

			der, _ := cmd.Flags().GetBool(DisableEmbeddedRollbacks)
			if der {
				options = options.DisableEmbeddedRollbacks()
			}

			migrations.Log.Infof("Looking for migrations in %s...", directory)
			if err := options.Apply(conn); err != nil {
				migrations.Log.Infof("Failed to migrate: %s", err)
				os.Exit(1)
			}

			migrations.Log.Infof("Migrations successfuly applied")
		},
	}

	cmd.Flags().String(Migrations, "./sql", "path to database migration (*.sql) files")
	cmd.Flags().String(URI, "", "URI to the PostgreSQL database, e.g. postgres://username:password@localhost:5432/database_name")
	cmd.Flags().Int(Revision, migrations.Latest, "migrate to this revision; defaults to latest")
	cmd.Flags().Bool(DisableEmbeddedRollbacks, false, "disable the embedded rollbacks functionality")

	return cmd
}

func getURI(cmd *cobra.Command) (string, error) {
	uri := os.Getenv("DB_URI")
	if uri != "" {
		return uri, nil
	}

	uri = os.Getenv("DB")
	if uri != "" {
		return uri, nil
	}

	return cmd.Flags().GetString(URI)
}

func main() {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "CLI for running database migrations",

		Run: func(cmd *cobra.Command, args []string) {
			version, _ := cmd.Flags().GetBool(Version)
			if version {
				fmt.Printf("Migrations CLI, v%s\n", CurrentVersion)
				return
			}

			_ = cmd.Help()
		},
	}

	cmd.Flags().BoolP(Version, "v", false, "version of the migrations library")

	cmd.AddCommand(createCmd())
	cmd.AddCommand(applyCmd())

	if err := cmd.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, err.Error())
		os.Exit(1)
	}
}
