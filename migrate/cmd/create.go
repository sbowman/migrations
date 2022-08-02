package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/sbowman/migrations"
)

// Create a migration file in the local directory.
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create new database migrations from the template",
	Long: `
The create command generates new SQL migration files in the migrations 
directory (./sql by default).  It will automatically generate the next 
version number for you.  

For example:

    $ migrate create create-users
    
`,

	Run: func(cmd *cobra.Command, args []string) {
		for _, arg := range args {
			if err := migrations.Create(viper.GetString(Migrations), arg); err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "Unable to create migration %s: %s", arg, err)
				os.Exit(1)
			}
		}
	},
}

func init() {
	root.AddCommand(createCmd)
}
