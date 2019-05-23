package cmd

import (
	"database/sql"
	"os"

	"github.com/sbowman/migrations"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Revision is the database migration revision setting name.
const Revision = "revision"

// Migrate the database.
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate the database",

	Run: func(cmd *cobra.Command, args []string) {
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
	conn, err := sql.Open(viper.GetString(Driver), viper.GetString(URI))
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
	dbCmd.AddCommand(migrateCmd)

	migrateCmd.Flags().Int(Revision, -1, "migrate to this revision; defaults to latest")

	viper.BindPFlag(Revision, migrateCmd.Flags().Lookup(Revision))
	viper.BindEnv(Revision, "REVISION")
}
