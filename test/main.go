package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	local "github.com/sbowman/migrations/cmd"
	remote "github.com/sbowman/migrations/remote/cmd"

	_ "github.com/lib/pq"
)

// URI of the PostgreSQL database for testing
const URI = "postgres://postgres@localhost/migrations?sslmode=disable"

func main() {
	viper.Set("uri", URI)

	root := &cobra.Command{
		Use:   "migrations",
		Short: "Migrations testing app",
	}

	root.PersistentFlags().Bool("remote", false, "enable remote S3 migrations for testing")

	testRemote := os.Getenv("TEST_REMOTE") == "true"
	if testRemote {
		remote.AddRemoteTo(root)
	} else {
		local.AddTo(root)
	}

	if err := root.Execute(); err != nil {
		fmt.Fprint(os.Stderr, "Failed to run: %s\n", err.Error())
	}
}
