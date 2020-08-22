package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	// URI is the PostgreSQL-compatible database URI used to connect to the database
	// to run migrations.
	URI = "uri"

	// Driver is the name of the database driver setting.
	Driver = "driver"

	// Migrations is the local database migrations path setting name.
	Migrations = "migrations"

	// Bucket is the AWS bucket name setting for the remote migrations.
	Bucket = "bucket"

	// Region is the AWS region name setting for the remote migrations.
	Region = "region"
)

// Database-related commands
var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Database migrations",
}

func init() {
	dbCmd.PersistentFlags().String(Migrations, "./sql", "local path to database migration (*.sql) files")
	dbCmd.PersistentFlags().String(Driver, "postgres", "name of the database driver")
	dbCmd.PersistentFlags().String(Bucket, "", "push and run migrations in this S3 bucket; leave blank to run local migrations")
	dbCmd.PersistentFlags().String(Region, "us-west-1", "the AWS region in which the bucket is located")

	_ = viper.BindPFlag(Migrations, dbCmd.PersistentFlags().Lookup(Migrations))
	_ = viper.BindPFlag(Driver, dbCmd.PersistentFlags().Lookup(Driver))
	_ = viper.BindPFlag(Bucket, dbCmd.PersistentFlags().Lookup(Bucket))
	_ = viper.BindPFlag(Region, dbCmd.PersistentFlags().Lookup(Region))

	_ = viper.BindEnv(Migrations, "MIGRATIONS")
	_ = viper.BindEnv(Driver, "DRIVER")
	_ = viper.BindEnv(Bucket, "BUCKET")
	_ = viper.BindEnv(Region, "REGION")
}

// AddRemoteTo applies the remote migration database commands under a "db"
// command.
//
// * db create - creates a new migration locally
// * db push - push the local migration files to S3
// * db migrate - runs the migrations from S3 bucket
//
// The "db migrate" command also accepts a "--local" flag to run migrations
// locally, from disk, instead of from the S3 bucket.  This allows testing of
// the migrations during development.
//
// Deprecated: to be removed in a future version.
func AddRemoteTo(cmd *cobra.Command) {
	cmd.AddCommand(dbCmd)
}
