package cli

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	// URI is the database URI used to connect to the database to run migrations.
	URI = "uri"

	// Driver is the name of the database driver setting.
	Driver = "driver"

	// Migrations is the name of the db.migrations spf13/viper setting.
	Migrations = "migrations"
)

// Database-related commands
var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Database migrations",
}

func init() {
	dbCmd.PersistentFlags().String(Migrations, "./sql", "path to database migration (*.sql) files")
	dbCmd.PersistentFlags().String(Driver, "postgres", "name of the database driver")

	_ = viper.BindPFlag(Migrations, dbCmd.PersistentFlags().Lookup(Migrations))
	_ = viper.BindPFlag(Driver, dbCmd.PersistentFlags().Lookup(Driver))

	_ = viper.BindEnv(Migrations, "MIGRATIONS")
	_ = viper.BindEnv(Driver, "DRIVER")
}

// AddTo applies the migration database commands under a "db" command.
//
// * db create - creates a new migration
// * db migrate - runs the migrations
//
// Deprecated: to be removed in a future version.
func AddTo(cmd *cobra.Command) {
	cmd.AddCommand(dbCmd)
}
