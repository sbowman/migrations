# Go SQL Migrations

[![PkgGoDev](https://pkg.go.dev/badge/github.com/sbowman/migrations)](https://pkg.go.dev/github.com/sbowman/migrations)

The migrations package is designed to help manage database revisions in Go
applications.  It works in a similar fashion to Ruby on Rails ActiveRecord 
migrations:  create a directory for your migrations, add numberically ordered
SQL files with an "up" and "down" section, then write your "up" and "down"
migrations.  You may then call functions in the package to migrate the database
up or down, based on the current "revision" of the database.

The "test" directory is used to test the migrations, both local and remote, and
provides a complete example of using the package.

Migrations 1.2.1 is completely API-compatible with Migrations 1.0.0.  Additionally,
all the new remote migration functionality and Cobra/Viper command support 
described below is completely optional.  It is isolated in packages so as not to 
pull in either the Cobra, Viper, or AWS packages into your binary unless you use
them.

## Deprecation Warnings

As of version 1.2, Cobra and Viper integrations are deprecated.  Obviously not
everyone is using these packages, and it seems presumptuous to force others to
include them.

The functionality will be completely removed in version 1.3.

## Adding Migrations to Your Application

There are a few primary functions to add to your applications to use the 
migrations package.

### Create an Empty Migration File

The migrations package can create empty migration files with the "up" and "down"
sections inserted for you (see below).

    migrations.Create("/path/to/migrations", "migration name")
    
This will create a migration file in the migrations path.  The name will be 
modified to remove the space, and a revision number will be inserted.  If there 
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
current version of your application.  Day to day, you'll likely just use the
value `-1`, which applies any and all existing migrations, in order of their
revision number.

But suppose you just want to apply the first four migrations to your database, 
but not the last two yet?  Set `revisionNum` to `4` and call `migrations.Migrate`.  
The migrations package will apply just the SQL from the migration files from 1 
to 4, in order.

What if you've already run all six migrations listed above?  When you call
`migrations.Migrate` with `revisionNum` set to `4`, the migrations package 
applies the SQL from the "down" section in migration `6-migration-name.sql`, 
followed by the SQL in the "down" section of `5-rename-sample-to-users.sql`.
We call this "rolling back" a migration, and it allows you to develop your 
application, make some changes, apply them, then roll them back if they don't
work.  

Note: *make sure to load the database driver before applying migrations!*

    import _ "github.com/lib/pq"
    
## Cobra & Viper

The migrations package includes sample commands that can be used as a model for 
how to wire up your own application.  You may also use them directly and quickly 
integrate migrations into an existing [https://github.com/spf13/cobra](Cobra) / 
[https://github.com/spf13/viper](Viper) based application.

To add the Cobra commands to your app, use something like the following:

    import migrations "github.com/sbowman/migrations/cmd"
    
    func init() {
        // ... other init code ...
        
        migrations.AddTo(RootCmd)
    }
  
If you do this, you don't have to mess with any of the main `migrations` 
package.  You'll automatically get a `db create` and a `db migrate` command
added you your application, with the following settings:

* `uri` - the database URI, e.g. "postgres://postgres@localhost/myapp"
* `driver` - the name of the database driver; defaults to. "postgres"
* `migrations` - the path to the migrations files, defaults to "./sql"
* `revision` - the revision number to run migrations, defaults to -1

## Local Migrations (1.0)

Typically you'll run migrations locally from disk, and either remotely provision
the database from your local machine, or copy the files with your application
deployment and provision your database directly on deployment or runtime from
your application server.

Create a directory in your application for the SQL migration files:

    $ mkdir ./sql
    $ cd sql

Now create a SQL migration.  The filename must start with a number, followed by
a dash, followed by some description of the migration.  For example:

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
this migration.  You may insert as many database commands as you like, but 
ideally each migration carries out the simplest, smallest unit of work that 
makes for a useful database, e.g. create a database table and indexes; make
a modification to a table; or create a stored procedure.

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
("up" or "down"), to the database.  The SQL calls are wrapped in a single 
transaction, so that if there's a problem with the SQL, the database can 
rollback the entire migration.  

Some databases, such as PostgreSQL, support nearly all schema modification 
commands (CREATE TABLE, ALTER TABLE, etc.) in a transaction.  Databases like 
MySQL have some support for this.  Your mileage may vary, but if your database 
doesn't support transactioned schema modifications, you may have to manually 
repair your databases should a migration partially fail.  This is another
reason to keep each migration modification small and targeted to a specific
change, and not put everything in one revision file:  if the migration fails
for whatever reason, it's easier to clean up.

## Remote S3 Migrations (1.1)

Migrations package version 1.1 adds support for "remote" migrations from the
Amazon AWS S3.  This allows you to upload your migrations to an S3 bucket, and
have your application apply the modifications from the S3 bucket instead of a
local disk.  This makes it easier to deploy something like AWS Lambda serverless
compute programs.

Currently remote migrations only support AWS S3.  Additional remote storage
systems may be supported in the future.

### AWS Credentials

The remote functionality assumes your AWS credentials are located in a 
`$HOME/.aws/credentials` file, similar to:

    [default]
    aws_access_key_id = <AWS access key>
    aws_secret_access_key = <AWS secret key> 

This may be customized to whatever security settings are necessary for your 
account.  See the [https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-files.html](S3 documentation) 
for more details.

### Enabling Remote Migrations

To configure migrations to work with S3, use the migrations package like normal,
but call `remote.InitS3`.  If you're using spf13/cobra, you might put it in a
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

### Cobra/Viper Support

Similar to the Cobra/Viper support describe above for migration commands against
local files, a set of remote S3-compatible commands exist that may be integrated
into your application.  These work nearly identially to the local versions, with
the addition of some S3 settings.

To add the remote Cobra commands to your app, use something like the following:

    import migrations "github.com/sbowman/migrations/remote/cmd"
    
    func init() {
        // ... other init code ...
        
        migrations.AddRemoteTo(RootCmd)
    }
  
If you do this, you don't have to mess with any of the main `migrations` or
`migrations/remote` packages.  You'll automatically get a `db create` and a `db
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
subdirectory.  The `db push` command will copy any new or updated files to the
S3 bucket for you:

    $ ./myapp db push --bucket="myapo-migrations"

You may put other files or directories in your bucket.  As long as they don't
end in `.sql`, the migrations will ignore any files or folders.

All other commands function in the same way as the local file versions, except
that you must supply a `--bucket` value:

    $ ./myapp db migrate --bucket="myapp-migrations" 

The supplied S3-based commands default to region `us-west-2`.  If your bucket
isn't in this region, supply a `--region=<name>` value for any command, e.g.
`--region=us-west-1`.

If you do not supply a bucket to the S3-based commands, the migrations package
applies the local migration files.  This can be useful for local development.
 
Additionally, you can run your migrations from within your app when it starts up:

    if err := migrations.Migrate(dbConn, "my-bucket-name", revisionNum); err != nil {
        return err
    }
    
Note that this is exactly the same as running on the local file system, except
the migration path is assumed to be the bucket name, "my-bucket-name".
   
## Migration Flags (1.3)

In version 1.3, migrations includes support for custom flags in the SQL scripts.
There is only one flag at present (`/notx`), but additonal flags could be 
added in the future.

To tweak how the migration is processed, include the flag at the end of the
up or down line/comment in the migration.  For example:

    # --- !Up /notx
    
Make sure to put a space between the direction value, e.g. `Up`, and the flag,
in this case, `/notx`

### Flag /notx

The `/notx` flag indicates to the migration processing functions that this
migration should **not** be run in a transaction.  

This is helpful in some situations, but use it with care and keep the `/notx` 
migrations small.  If the migration only partially completes, you may need to 
manually clean up the database before migrations can continue.
