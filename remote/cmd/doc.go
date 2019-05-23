// The cmd package for remote migrations provides some github.com/spf13/cobra
// commands that may be shared and used in components using the migrations
// remote package.
//
// For example, simply add the commands to your root command:
//
//     import migrations "github.com/sbowman/migrations/remote/cmd"
//
//     func init() {
//         // ... configure other app settings
//
//         migrations.AddRemoteTo(RootCmd)
//     }
//
// The run the "db create" or "db migrate" commands:
//
//     $ myapp db migrate --revision=12 --region=us-west-2 \
//     --bucket=myapp-migrations
//
// This is entirely optional, and provided as a separate package so as not to
// taint your imports with the spf13 cobra and viper projects if you're not
// using them.
package cmd
