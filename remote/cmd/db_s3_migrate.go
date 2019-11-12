package cmd

import (
	"database/sql"
	"os"

	"github.com/sbowman/migrations/v2/remote"

	"github.com/sbowman/migrations/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Revision is the migration revision number setting.
const Revision = "revision"

// Migrate the database.
var migrateS3Cmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate the database",

	Run: func(cmd *cobra.Command, args []string) {
		if viper.GetString(Bucket) != "" {
			migrations.Log.Infof("Running remote migrations in region %s, bucket %s", viper.GetString(Region), viper.GetString(Bucket))

			// This is all that's required to do to run migrations from S3 buckets
			if err := remote.InitS3(viper.GetString(Region)); err != nil {
				migrations.Log.Infof("Unable to connect to S3: %s", err)
				os.Exit(1)
			}
		}

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

	// Because we identify the local migrations directory (db.migrations)
	// and the S3 bucket differently (db.bucket), we should check local vs.
	// remote migrations...
	location := viper.GetString(Bucket)
	if location == "" {
		location = viper.GetString(Migrations)

		// Temporarily use disk-based migrations
		current := migrations.IO
		migrations.IO = new(migrations.DiskReader)
		defer func() {
			migrations.IO = current
		}()
	}

	migrations.Log.Infof("Running migrations in %s...", location)
	if err := migrations.Migrate(conn, location, viper.GetInt(Revision)); err != nil {
		migrations.Log.Infof(err.Error())
		os.Exit(1)
	}

	return nil
}

func init() {
	dbCmd.AddCommand(migrateS3Cmd)

	migrateS3Cmd.Flags().Int(Revision, -1, "migrate to this revision; defaults to latest")

	viper.BindPFlag(Revision, migrateS3Cmd.Flags().Lookup(Revision))
	viper.BindEnv(Revision, "REVISION")
}
