// Remote package allows for migrations to be run from a remote source, such as
// S3.
//
// To configure migrations to work with S3, use the migrations package like
// normal, but call `remote.InitS3`.  If you're using spf13/cobra, you might
// put it in a PersistentPreRun function:
//
//     rootCmd = &cobra.Command{
//         PersistentPreRun: func(cmd *cobra.Command, args []string) {
//             remote.InitS3(viper.GetString("s3.region"))
//         },
//     // ...
//
// Make sure your S3 credentials are defined in ~/.aws/credentials, per the
// Amazon AWS instructions.
//
// You may copy the SQL migrations to an S3 bucket manually, or use the S3 push
// function:
//
//     if err := remote.PushS3(viper.GetString("migrations),
//             viper.GetString("region"), viper.GetString("bucket")); err != nil {
//         fmt.Fprintf(os.Stderr, "Failed to push migrations to S3: %s\n", err)
//         os.Exit(1)
//     }
//
// The default "db create" command creates migrations in a local directory, in
// expectation you'll use a "db push" command based on the above to push the
// migrations to S3.
//
// After the InitS3 call, you can run the remote migrations the same way you
// run standard, disk-based migrations, but pass in the bucket name instead of
// a migrations directory:
//
//     conn, err := sql.Open("postgres", "postgres://....")
//     if err != nil {
//         return err
//     }
//
//     if err := migrations.Migrate(conn, viper.GetString("bucket"),
//             viper.GetInt("revision")); err != nil {
//         fmt.Fprintf(os.Stderr, "Failed to migrate: %s\n", err)
//         os.Exit(1)
//     }
//
// See the remote/cmd package for examples (or feel free to use them in your
// own spf13/cobra and spf13/viper applications).
package remote
