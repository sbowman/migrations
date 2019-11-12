package cmd

import (
	"os"

	"github.com/sbowman/migrations/v2"
	"github.com/sbowman/migrations/v2/remote"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Push local migrations to the remote S3 bucket.
var pushS3Cmd = &cobra.Command{
	Use:   "push",
	Short: "Push new and updated local migration files to the S3 bucket",

	Run: func(cmd *cobra.Command, args []string) {
		migrations.Log.Infof("Pushing local migrations from %s to region %s, bucket %s",
			viper.GetString(Migrations), viper.GetString(Region), viper.GetString(Bucket))

		if err := remote.PushS3(viper.GetString(Migrations), viper.GetString(Region), viper.GetString(Bucket)); err != nil {
			migrations.Log.Infof("Failed to push to S3: %s", err)
			os.Exit(1)
		}
	},
}

func init() {
	dbCmd.AddCommand(pushS3Cmd)
}
