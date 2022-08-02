package cmd

import (
	"os"

	"github.com/sbowman/migrations"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Create a migration file in the local directory.
var rollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Rollback the database to the ",

	Run: func(cmd *cobra.Command, args []string) {
		for _, arg := range args {
			if err := migrations.Create(viper.GetString(Migrations), arg); err != nil {
				migrations.Log.Infof(err.Error())
				os.Exit(1)
			}
		}
	},
}

func init() {
	root.AddCommand(createCmd)
}
