package cmd

import (
	"os"

	"github.com/sbowman/migrations/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Create a migration file in the local directory.
var dbCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create new database migrations from the template",

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
	dbCmd.AddCommand(dbCreateCmd)
}
