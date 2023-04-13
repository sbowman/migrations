# Go SQL Migrations v2.0.0

## TODO

[![PkgGoDev](https://pkg.go.dev/badge/github.com/sbowman/migrations/v2)](https://pkg.go.dev/github.com/sbowman/migrations/v2)

The `migrations` package provides an agile approach to managing database schema revisions for your
Go applications. Using versioned files containing SQL commands to modify the schema, you can make
small changes to your database schema over time and version the changes using version controller.

The Migrations package is designed to help manage PostgreSQL database schema revisions in Go
applications. It works in a similar fashion to Ruby on Rails ActiveRecord migrations:  create a
directory for your migrations, add numberically ordered SQL files with an "up" and "down" section,
then write your "up" SQL commands to update the database, and "down" SQL command to roll back
those changes if needed. You may then call functions in the package from your application to apply
changes to the database schema on deployment, or leverage the `migrate` command line application
to manage the database schema separately from the application.

Version 2 of the Migrations package trims down a lot of functionality that had crept into the
previous version. Note that asynchronous migrations, Cobra/Viper integration, remote S3 migrations,
and transactionless migrations have all been removed due to lack of use. If you'd like them added,
feel free to post a ticket.

## Upgrading From Migrations/v1

If your application was using the v1 version of the `migrations` package, it will automatically be
upgraded to v2. The following changes are made to the database:

* A new schema, `migrations` is created in the database.
* Two tables are created in this schema, `migrations.applied` and `migrations.rollbacks`.
* The migrations from the `schema_migrations` table are copied to `migrations.applied`.
* The `schema_migrations` table is deleted.

Migrations metadata is now maintained in the `migrations` schema in an attempt to keep things
separate from your database and out of your way. The `migrations.applied` table is the old
`schema_migrations` table.

The `migrations.rollbacks` is used to store rollback or "down" SQL migrations in the database.  
This allows an application to rollback the database when downgrading an application and the SQL
files to rollback the database schema are no longer to the application.

## Migrations Integration and CLI

The `migrations` package may be included in your product binary, or run separately as a standalone
application. When embedded in the application, typically you configure your application to apply the
database migrations on startup.

It may also be run as a standalone tool, either from the command line or a Docker container.

### Installing the Command-Line Tool

To install the command-line tool, run:

    $ go install github.com/sbowman/migrations/v2/migrate

This will install the `migrate` binary into your `$GOPATH/bin` directory. If that directory is on
your PATH, you should be able to run migrations from anywhere.

If you need help using the command-line tool, run `migrate help` for information.

## Available Options

To apply migrations to the database schema, you need to supply two pieces of information:  the path
to the SQL migrations, and the database URI to connect to the database. To supply that information
to the command-line application, use the `migrations` and `uri` parameters:

    $ migrate apply --uri='postgres://postgres@127.0.0.1/fourier?sslmode=disable&connect_timeout=5'
    --migrations=./sql

By default, the `migrations` package looks for migrations in the `sql` directory relative to
where you run the binary, so you can leave `--migrations` off:

    $ migrate apply --uri='postgres://postgres@127.0.0.1/fourier?sslmode=disable&connect_timeout=5'

You may additionally apply a revision to migrate the database to the specific version of the schema
using the `--revision` parameter:

    $ migrate apply --uri='postgres://postgres@127.0.0.1/fourier?sslmode=disable&connect_timeout=5'
    --revision=23

This will either apply the migrations to reach SQL migration file `23-<name>.sql`, or rollback the
database migrations to step back down to revision 23.

### Supported Environment Variables

The `migrations` package supports supplying configuration options via environment variables.

* To supply an alternative migrations directory, you can use the `MIGRATIONS` environment variable.
* To supply the PostgreSQL URI in the command-line application, you can use the `DB_URI` or `DB`
  environment variables.

## Development and Testing Notes

In order to remove the PostgreSQL driver dependency from this package, test cases exist
in a subdirectory, `tests`, and are treated as a standalone package. This allows
us to maintain test cases, but utilize the PostgreSQL driver to test `migrations`
against a real database server.

To run these test cases:

    cd tests && make

The `Makefile` tears down any existing `migrations_test` database, creates a new
`migrations_test` database, runs the tests for migrations in a transaction,
followed by tests for migrations outside of a transaction.

## The API

### Adding Migrations to Your Application

There are a few primary functions to add to your applications to use the `migrations` package.

### Create an Empty Migration File

The migrations package can create empty migration files with the "up" and "down" sections inserted
for you (see below).

    migrations.Create("/path/to/migrations", "migration name")

This will create a migration file in the migrations path. The name will be modified to remove the
space, and a revision number will be inserted. If there were five other SQL migrations in
`/path/to/migrations`, You'd end up with a file like so:

    $ ls /path/to/migrations
    total 16
    -rw-r--r--  1 sbowman  staff   99 May 22 15:39 1-create-sample.sql
    -rw-r--r--  1 sbowman  staff  168 May 22 15:40 2-add-email-to-sample.sql
    -rw-r--r--  1 sbowman  staff  207 May 22 15:50 3-create-my-proc.sql
    -rw-r--r--  1 sbowman  staff   96 May 22 16:32 4-add-username-to-sample.sql
    -rw-r--r--  1 sbowman  staff  347 May 22 17:55 5-rename-sample-to-users.sql
    -rw-r--r--  1 sbowman  staff   22 May 26 19:03 6-migration-name.sql

### Apply the Migrations

Once you've created your migration files and added the appropriate SQL to generate your database
schema and seed any data, you'll want to apply the migrations to the database by calling:

    migrations.Apply(conn)

Where `conn` is a *sql.Conn database connection (from the `database/sql` Go package).

This will attempt to run the migrations to the latest version as defined in the default `./sql`
directory, relative to where the binary was run.

To indicate another directory:

    migrations.WithDirectory("/etc/app/sql").Apply(conn)

To migrate to a specific revision:

    migrations.WithDirectory("/etc/app/sql").WithRevision(33).Apply(conn)

The revision number allows you to apply just the migrations appropriate for the current version of
your application. Day to day, you'll likely just use the default value `-1`, which applies any and
all existing migrations, in order of their revision number.

But suppose you just want to apply the first four migrations to your database, but not the last two
yet? Set `revisionNum` to `4` and call `migrations.Migrate`. The migrations package will apply
just the SQL from the migration files from 1 to 4, in order.

What if you've already run all six migrations listed above? When you call `migrations.Apply` with
`WithRevision` set to `4`, the migrations package applies the SQL from the "down" section in
migration `6-migration-name.sql`, followed by the SQL in the "down" section of
`5-rename-sample-to-users.sql`.

We call this "rolling back" a migration, and it allows you to develop your application, make some
changes, apply them, then roll them back if they don't work.

Note: *make sure to load the database driver before applying migrations!*  For
example:

    import _ "github.com/jackc/pgx/v5/stdlib"

See `tests/migrations_test.go` for an example.

## Migration Files

Typically you'll deploy your migration files to a directory when you deploy your
application.

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

    --- !Up
    
    --- !Down

Under the "up" section, add the changes you'd like to make to the database in
this migration. You may insert as many database commands as you like, but
ideally each migration carries out the simplest, smallest unit of work that
makes for a useful database, e.g. create a database table and indexes; make
a modification to a table; or create a stored procedure.

Note the above line formats are required, including the exclamation points, or
`migrations` won't be able to tell "up" from "down."

The "down" section should contain the code necessary to rollback the "up"
changes.

So our "create_users" migration may look something like this:

    --- !Up
    create table users (
        id serial primary key,
        username varchar(64) not null,
        email varchar(1024) not null,
        password varchar(40) not null,
        enabled bool not null default true
    );
    
    create unique index idx_enabled_users on users (username) where enabled;
    create unique index idx_enabled_emails on users (email) where enabled;
    
    --- !Down
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

## Using the Migrations command-line tool

The Migrations v2 includes a CLI tool to run migrations standalone, without
needing to embed them in your application. Simply install the CLI tool and
make sure the Go `bin` directory is in your path:

    $ go install github.com/sbowman/migrations/v2/migrate
    $ migrate --revision=12 --migrations=./sql --uri=postgres://localhost/myapp_db?sslmode=disable 

Use `migrate --help` for details on the available commands and parameters.

## Embedded Rollbacks

Migrations/v2 stores each rollback ("down") SQL migration in the database. With this the migrations
package doesn't need the SQL files to be present to rollback, which makes it easier to rollback an
application's database migrations when using deployment tools like Ansible or Terraform. You can
simply deploy a previous version of the application, and the migrations package can apply the
rollbacks stored in the database to restore the database to its previous schema version.

For example, you could deploy version 1.3 of your application, realize there is a bug, then
redeploy version 1.2. The migrations package can recognize the highest version of SQL files
available is lower than the migrations applied to the database, and can run the rollback using the
SQL embedded in the database table `migrations.rollbacks.`

The migrations package manages this with no additional work on the developer's part.

### The /stop Annotation

Some migrations can't be rolled back. For example, if you delete data from the database, you're
not going to be able to rollback and restore that data. If you'd like to indicate a migration
can't be rolled back, you can use the `/stop` annotation:

    --- !Up
    <some irreversible SQL>

    --- !Down /stop

By adding `/stop` to the "down" migration, you indicate to the `migrations` library that this
migration cannot be rolled back and a rollback will stop when it reaches this migration. Without
this flag, the "down" migration would simply be skipped for a lack of SQL and the rollback would
continue.

Use this functionality sparingly. Migrations should be small, and they should be reversable. If
you do need to make an irreversible change to a database, best to take a deprecation step:  create
the schema changes, but keep the old schema (and data) around for a while, then finally deprecate
in a subsequent, future revision.

## Logging

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

