package cmd

import (
	"database/sql"
	"os"

	"github.com/sbowman/migrations"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	_ "github.com/jackc/pgx/v4/stdlib"
)

const (
	// URI is the database URI used to connect to the database to run migrations (`--uri`).
	URI = "uri"

	// Migrations is the name of the db.migrations spf13/viper setting (`--migrations`).
	Migrations = "migrations"

	// Revision is the database migration revision setting name (`--revision`); ignored if
	// `--auto` is present.
	Revision = "revision"

	// Auto flag indicates to use the SQL migration file numbers as the revision (`--auto`);
	// `--revision` is ignored when `--auto` is used.
	Auto = "auto"
)

var root = &cobra.Command{
	Use:   "migrate",
	Short: "Runs PostgreSQL database migrations",

	Run: func(_ *cobra.Command, _ []string) {
		// TODO:  Add check for migrating to the right revision using schema_rollbacks

		if viper.GetInt(Revision) >= 0 {
			migrations.Log.Infof("Migrating %s to revision %d", viper.GetString(URI), viper.GetInt(Revision))
		} else {
			migrations.Log.Infof("Migrating %s to the latest revision", viper.GetString(URI))
		}

		if err := runMigrations(); err != nil {
			migrations.Log.Infof("Failed to migrate: %s", err)
		}

	},
}

func runMigrations() error {
	conn, err := sql.Open("pgx", viper.GetString(URI))
	if err != nil {
		return err
	}

	migrations.Log.Infof("Running migrations in %s...", viper.GetString(Migrations))
	if err := migrations.Migrate(conn, viper.GetString(Migrations), viper.GetInt(Revision)); err != nil {
		migrations.Log.Infof(err.Error())
		os.Exit(1)
	}

	return nil
}

func init() {
	root.PersistentFlags().String(URI, "", "the database connection URI, e.g. postgres://username:password@localhost:5432/database_name")
	root.PersistentFlags().String(Migrations, "./sql", "path to database migration (*.sql) files")
	root.Flags().Int(Revision, -1, "migrate to this revision; defaults to latest")
	root.Flags().Bool(Auto, false, "migrate or rollback to the highest SQL migration file number")

	_ = viper.BindPFlag(URI, root.PersistentFlags().Lookup(URI))
	_ = viper.BindPFlag(Migrations, root.PersistentFlags().Lookup(Migrations))
	_ = viper.BindPFlag(Revision, root.Flags().Lookup(Revision))
	_ = viper.BindPFlag(Auto, root.Flags().Lookup(Auto))

	_ = viper.BindEnv(URI, "DB_URI")
	_ = viper.BindEnv(Migrations, "MIGRATIONS")
	_ = viper.BindEnv(Revision, "REVISION")
	_ = viper.BindEnv(Auto, "AUTOMIGRATE")
}
