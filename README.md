# Go SQL Migrations

[![PkgGoDev](https://pkg.go.dev/badge/github.com/sbowman/migrations)](https://pkg.go.dev/github.com/sbowman/migrations)

See https://github.com/sbowman/migrations/blob/master/v2/README.md for `migrations/v2`.

The migrations package is designed to help manage database revisions in Go
applications. It works in a similar fashion to Ruby on Rails ActiveRecord
migrations:  create a directory for your migrations, add numberically ordered
SQL files with an "up" and "down" section, then write your "up" and "down"
migrations. You may then call functions in the package to migrate the database
up or down, based on the current "revision" of the database.

The `migrations` package is designed to be included in your product binary.
When the application deploys to the server, you can run your migrations from
the server, without manually having to setup the database.

Alternatively, you could build a small CLI binary or Docker container, include
the `migrations` package and your SQL files in the image, and run that
standalone.

In a cloud environment like Kubernetes, you might create a "maintenance" pod
to run migrations on startup (and periodically clean out old data from the
database). Implement an HTTP endpoint that holds on to any clients until
`migrations` complete. When a client starts, it requests that maintentance HTTP
endpoint before accepting its own requests. When the migrations finish in the
maintenance pod, the maintenance pod allows the HTTP endpoint to finally reply
(and simply reply with a 2XX immediately for any future requests). This ensures
your applications, on deployment, all wait for migrations to complete before
accepting requests, so they are using the latest and greatest version of your
database.

Migrations 1.4.0 is completely API-compatible with Migrations 1.0.0.

All the remote migration functionality and Cobra/Viper command support described
below is completely optional. It is isolated in packages so as not to pull in
either the Cobra, Viper, or AWS packages into your binary unless you use them.

## Development Notes

The "sql" directory is used to test the migrations. Its contents should *NOT*
be included in your applications.

The "cmd" directory contains some sample code describing how to setup
`migrations` functionaltiy in your applications, using the Cobra and Viper
packages.  (_This package will eventually be deprecated, and the examples moved
to proper `Example` code._)

## Testing

In order to remove the PostgreSQL dependency from this package, as of 1.4 test
cases have been moved to the `tests` directory, and are treated as a standalone
package. This allows us to maintain test cases, but utilize the PostgreSQL
driver to test `migrations` against a real database server.

To run these test cases:

    cd tests && make

The `Makefile` tears down any existing `migrations_test` database, creates a new
`migrations_test` database, runs the tests for migrations in a transaction,
followed by tests for migrations outside of a transaction.

## Deprecation Warnings

Version 1.2 deprecates Cobra and Viper integrations. Obviously not everyone is
using these packages, and it seems presumptuous to force others to include them.

The functionality will be completely removed in version 1.5.

## Adding Migrations to Your Application

There are a few primary functions to add to your applications to use the
migrations package.

### Create an Empty Migration File

The migrations package can create empty migration files with the "up" and "down"
sections inserted for you (see below).

    migrations.Create("/path/to/migrations", "migration name")

This will create a migration file in the migrations path. The name will be
modified to remove the space, and a revision number will be inserted. If there
were five other SQL migrations in `/path/to/migrations`, You'd end up with a
file like so:

    $ ls /path/to/migrations
    total 16
    -rw-r--r--  1 sbowman  staff   99 May 22 15:39 1-create-sample.sql
    -rw-r--r--  1 sbowman  staff  168 May 22 15:40 2-add-email-to-sample.sql
    -rw-r--r--  1 sbowman  staff  207 May 22 15:50 3-create-my-proc.sql
    -rw-r--r--  1 sbowman  staff   96 May 22 16:32 4-add-username-to-sample.sql
    -rw-r--r--  1 sbowman  staff  347 May 22 17:55 5-rename-sample-to-users.sql
    -rw-r--r--  1 sbowman  staff   22 May 26 19:03 6-migration-name.sql

### Apply the Migrations

Once you've created your migration files and added the appropriate SQL to
generate your database schema and seed any data, you'll want to apply the
migrations to the database by calling:

    migrations.Migrate(dbConn, "/path/to/migrations", revisionNum) 

Where `dbConn` is a *sql.Conn database connection (from the `database/sql` Go
package), and `revisionNum` is the revision number.

The revision number allows you to apply just the migrations appropriate for the
current version of your application. Day to day, you'll likely just use the
value `-1`, which applies any and all existing migrations, in order of their
revision number.

But suppose you just want to apply the first four migrations to your database,
but not the last two yet? Set `revisionNum` to `4` and call `migrations.Migrate`.  
The migrations package will apply just the SQL from the migration files from 1
to 4, in order.

What if you've already run all six migrations listed above? When you call
`migrations.Migrate` with `revisionNum` set to `4`, the migrations package
applies the SQL from the "down" section in migration `6-migration-name.sql`,
followed by the SQL in the "down" section of `5-rename-sample-to-users.sql`.
We call this "rolling back" a migration, and it allows you to develop your
application, make some changes, apply them, then roll them back if they don't
work.

Note: *make sure to load the database driver before applying migrations!*  For
example:

    import _ "github.com/jackc/pgx/v4/stdlib"

See `tests/migrations_test.go` for an example.

## Migrate app (1.5)

The `migrations` package used to include integration with Cobra and Viper. This
was deprecated in 1.4 and is now officially removed. In its place is a sample
application, `migrate`, leveraging Cobra and Viper to demonstrate using the
`migrations` package in an application.

For PostgreSQL users, this sample app may also be installed as a standalone app
to run migrations and create SQL migration file templates.

The following command-line parameters/environment variables are supported by the
`migrate` application:

* `uri`/`DB_URI` - the database URI, e.g.
  `postgres://username:password@localhost:5432/database_name`
* `migrations`/`MIGRATIONS` - the path to the migrations files, defaults to "./sql"
* `revision`/`REVISION` - the revision number to run migrations, defaults to latest
* `auto`/`AUTOMIGRATE` - migrate or rollback the database to the version matching
  the available SQL migrations

With `--auto`, the `migrate` app will use the SQL migration files' latest version
as the version to migrate to. If the database is at a later version and supports
`schema_rollbacks`, a `migrate --auto` will rollback the database to that version.
Otherwise it will migrate to the latest SQL migration file revision.

### Embedded Rollbacks (1.5)

The `migrations` package supports optionally including the SQL rollback (`Down`)
section of each migration in a new database table, `schema_rollbacks`. This
allows migrations to be rolled back without requiring the original SQL files.
This can be useful in environments such as Docker and Kubernetes when you need
to rollback a version to a previous image or pod, which means the SQL files to
rollback the database migrations won't be present. When present, these embedded
SQL commands can be used to rollback the database to the correct version for the
image now in use.

If this information is not in the database and the database requires a rollback,
you may still use the SQL files to perform the rollback. You'll just have to do
that manually.

Additionally, the `migrations` package now has a `Check()` function to check the
version of the SQL migration files versus what migrations have been applied to
the database. This function returns `nil` if the current SQL migrations have all
been applied to the database, `ErrMigrationRequired` if there are unapplied
migrations, or `ErrRollbackRequired` if there are migrations applied to the
database that are no longer present in the SQL migration files.

### Transactionless Migrations (1.4)

As of version 1.4, the `migrations` package adds support for running migrations
outside of a transaction. This package was originally designed to work with
PostgreSQL, which supports DDL commands inside of a transaction. But this
feature is not available in all databases, and running statements inside a
transaction on some of these databases can cause errors. As of version 1.4,
you may run migrations in transactionless mode.

Transactionless migrations are obviously less safe than running migrations in
a transaction. If you have migrations with multiple SQL commands, and a
command fails in the middle of the migration, your database will be in a
partially migrated state and you'll have to clean up the migration manually to
continue with the migration. If you use `migrations` like this, each individual
SQL migration file should contain the minimum number of SQL commands as is
reasonable, so that if a migration fails it's easier to resolve.

To run transactionless migrations, call `MigrateUnsafe` instead of `Migrate`.

### Asynchronous Migrations (1.3)

As of version 1.3, the `migrations` package supports asynchronous migrations
(see the `/async` flag below). This allows longer running migrations to run
in the background, while allowning the main, synchronous migrations to complete.

If you use the `Migrate` function, asynchronous migrations are ignored and run
like normal migrations. This is useful in development. To run asynchronous
migrations in the background, use `MigrateAsync`

For example, if you're using something like the "maintenance" pod described in
the introduction, you may want to allow your migrations to run to completion,
let your other pods depending on those synchronous migrations know the database
is ready, while letting the longer running asynchronous migrations more time to
complete in the background. To do this, call `MigrateAsync` and handle the
`migrations.ResultChannel` yourself.

Here's some sample code illustrating `MigrateAsync`:

	asyncResults, err := migrations.MigrateAsync(dbConn, pathToMigrations, 
	    migrateToRevision)
	if err != nil {
		return fmt.Errorf("migrations failed: %s', err)
	}

    // Once migrations.MigrateAsync returns, all the synchronous migrations 
    // have completed...
    NotifyClientsMigrationsCompleted();
    
	// Blocks until the asynchronous requests complete; logs any errors in the
	// asynchronous migrations 
	for result := range asyncResults {
		if result.Err != nil {
			if result.Command == "" {
				Log.Infof("Asynchronous migration %s failed: %s", 
				    result.Migration, result.Err)
				continue
			}

			Log.Infof("Asynchronous migration %s failed on command %s: %s",
				result.Migration, result.Command, result.Err)
		}
	}

    // When asynchronous migrations finish, the asyncResults channel closes

## Logging (1.0)

The `migrations` package uses a simple `Logger` interface to expose migration
information to the user. By default, this goes to `stdout`. You're welcome to
implement your own logger and wire migrations logging into your own log output.

Here's the interface:

    // Logger is a simple interface for logging in migrations.
    type Logger interface {
        Debugf(format string, args ...interface{})
        Infof(format string, args ...interface{})
    }

Just assign your logger to `migrations.Log` before running any migration
functions.

There's also a `NilLogger` available, if you'd like to hide all `migrations`
output.

    migrations.Log = new(migrations.NilLogger)

## Local Migrations (1.0)

Typically you'll run migrations locally from disk, and either remotely provision
the database from your local machine, or copy the files with your application
deployment and provision your database directly on deployment or runtime from
your application server.

Create a directory in your application for the SQL migration files:

    $ mkdir ./sql
    $ cd sql

Now create a SQL migration. The filename must start with a number, followed by
a dash, followed by some description of the migration. For example:

    $ vi 1-create_users.sql

If you're using the Cobra commands, there's a "db create" function that creates
the directory and numbers the migration file correctly for you (if migrations
already exist, it'll figure out the next number):

    $ ./myapp db create create_users
    Created new migration ./sql/1-create_users.sql

An empty migration file looks like this:

    # --- !Up
    
    # --- !Down

Under the "up" section, add the changes you'd like to make to the database in
this migration. You may insert as many database commands as you like, but
ideally each migration carries out the simplest, smallest unit of work that
makes for a useful database, e.g. create a database table and indexes; make
a modification to a table; or create a stored procedure.

Note the above line formats are required, including the exclamation points, or
`migrations` won't be able to tell "up" from "down."  **As of version 1.3, the
`#` at the front of the up and down sections is optional, e.g. `--- !Up` is
equivalent to `# --- !Up`, to support a valid SQL syntax and syntax
highlighters.**

The "down" section should contain the code necessary to rollback the "up"
changes.

So our "create_users" migration may look something like this:

    # --- !Up
    create table users (
        id serial primary key,
        username varchar(64) not null,
        email varchar(1024) not null,
        password varchar(40) not null,
        enabled bool not null default true
    );
    
    create unique index idx_enabled_users on users (username) where enabled;
    create unique index idx_enabled_emails on users (email) where enabled;
    
    # --- !Down
    drop table users;

The migrations package simply passes the raw SQL in the appropriate section
("up" or "down"), to the database. The SQL calls are wrapped in a single
transaction, so that if there's a problem with the SQL, the database can
rollback the entire migration (_see below to disable these transactions for
special cases_).

Some databases, such as PostgreSQL, support nearly all schema modification
commands (`CREATE TABLE`, `ALTER TABLE`, etc.) in a transaction. Databases like
MySQL have some support for this. Your mileage may vary, but if your database
doesn't support transactioned schema modifications, you may have to manually
repair your databases should a migration partially fail. This is another
reason to keep each migration modification small and targeted to a specific
change, and not put everything in one revision file:  if the migration fails
for whatever reason, it's easier to clean up.

## Remote S3 Migrations (1.1)

Migrations package version 1.1 adds support for "remote" migrations from the
Amazon AWS S3. This allows you to upload your migrations to an S3 bucket, and
have your application apply the modifications from the S3 bucket instead of a
local disk. This makes it easier to deploy something like AWS Lambda serverless
compute programs.

Currently remote migrations only support AWS S3. Additional remote storage
systems may be supported in the future.

### AWS Credentials

The remote functionality assumes your AWS credentials are located in a
`$HOME/.aws/credentials` file, similar to:

    [default]
    aws_access_key_id = <AWS access key>
    aws_secret_access_key = <AWS secret key> 

This may be customized to whatever security settings are necessary for your
account. See
the [https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-files.html](S3 documentation)
for more details.

### Enabling Remote Migrations

To configure migrations to work with S3, use the migrations package like normal,
but call `remote.InitS3`. If you're using spf13/cobra, you might put it in a
PersistentPreRun function:

    rootCmd = &cobra.Command{
        PersistentPreRun: func(cmd *cobra.Command, args []string) {
            remote.InitS3(viper.GetString("s3.region"))
        },
    // ...

### Pushing Migrations

You may copy the SQL migrations to an S3 bucket manually, or use the S3 push
function:

    if err := remote.PushS3(viper.GetString("migrations),
            viper.GetString("region"), viper.GetString("bucket")); err != nil {
        fmt.Fprintf(os.Stderr, "Failed to push migrations to S3: %s\n", err)
        os.Exit(1)
    }

The default "db create" command creates migrations in a local directory, in
expectation you'll use a "db push" command based on the above to push the
migrations to S3.

### Applying Remote Migrations

After the InitS3 call, you can run the remote migrations the same way you run
standard, disk-based migrations, but pass in the bucket name instead of a
migrations directory:

    conn, err := sql.Open("postgres", "postgres://....")
    if err != nil {
        return err
    }

    if err := migrations.Migrate(conn, viper.GetString("bucket"),
            viper.GetInt("revision")); err != nil {
        fmt.Fprintf(os.Stderr, "Failed to migrate: %s\n", err)
        os.Exit(1)
    }

See the remote/cmd package for examples (or feel free to use them in your own
spf13/cobra and spf13/viper applications).

### Cobra/Viper Support (Deprecated)

Similar to the Cobra/Viper support describe above for migration commands against
local files, a set of remote S3-compatible commands exist that may be integrated
into your application. These work nearly identially to the local versions, with
the addition of some S3 settings.

To add the remote Cobra commands to your app, use something like the following:

    import migrations "github.com/sbowman/migrations/remote/cmd"
    
    func init() {
        // ... other init code ...
        
        migrations.AddRemoteTo(RootCmd)
    }

If you do this, you don't have to mess with any of the main `migrations` or
`migrations/remote` packages. You'll automatically get a `db create` and a `db
migrate` command, like the local commands, but you'll also get a third command,
`db push`, described below.

The following settings are supported with the S3 remote Cobra commands:

* `uri` - the database URI, e.g. "postgres://postgres@localhost/myapp"
* `driver` - the name of the database driver; defaults to. "postgres"
* `migrations` - the path to the migrations files, defaults to "./sql"
* `revision` - the revision number to run migrations, defaults to -1
* `region` - the AWS region the bucket is in; defaults to "us-west-2"
* `bucket` - the name of the bucket holding the migration files

You should copy your migrations directly into the bucket; do not put them in a
subdirectory. The `db push` command will copy any new or updated files to the
S3 bucket for you:

    $ ./myapp db push --bucket="myapo-migrations"

You may put other files or directories in your bucket. As long as they don't
end in `.sql`, the migrations will ignore any files or folders.

All other commands function in the same way as the local file versions, except
that you must supply a `--bucket` value:

    $ ./myapp db migrate --bucket="myapp-migrations" 

The supplied S3-based commands default to region `us-west-2`. If your bucket
isn't in this region, supply a `--region=<name>` value for any command, e.g.
`--region=us-west-1`.

If you do not supply a bucket to the S3-based commands, the migrations package
applies the local migration files. This can be useful for local development.

Additionally, you can run your migrations from within your app when it starts up:

    if err := migrations.Migrate(dbConn, "my-bucket-name", revisionNum); err != nil {
        return err
    }

Note that this is exactly the same as running on the local file system, except
the migration path is assumed to be the bucket name, "my-bucket-name".

## Migration Flags (1.3)

In version 1.3, migrations includes support for custom flags in the SQL scripts.
There is only one flag at present (`/async`), but additonal flags could be
added in the future.

To tweak how the migration is processed, include the flag at the end of the
up or down line/comment in the migration. For example:

    # --- !Up /async

Make sure to put a space between the direction value, e.g. `Up`, and each flag,
in this case, `/async`.

### Flag /async

The `/async` flag will run the migration in a separate thread. This can be
useful for running migrations that can optionally fail, or long-running
migrations that can run in the background for a while after your application
starts.

Use this flag with caution. The migration will be marked as completed (recorded
in the `schema_migrations` table in the database) immediately after it is
handed off to the background process. If it fails, it will return an error on
a results channel (which your application may listen for), but it will still
appear to be successfully completed (because it's in the `schema_migrations`
table). This is designed this way on purpose, to be the least invasive
approach to asynchronous migrations. But that means if something goes wrong
in an asynchronous migration, it's your responsibility to manually resolve the
problem.

To use the `/async` function there is a new `MigrateAsync` function. See the
example above for a sample of how to use it. If you don't call `MigrateAsync`,
the `/async` flag is ignored and the migration run as a normal migration.
