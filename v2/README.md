# Go SQL Migrations v2.0.0

[![PkgGoDev](https://pkg.go.dev/badge/github.com/sbowman/migrations/v2)](https://pkg.go.dev/github.com/sbowman/migrations/v2)

Migrations v2 is designed specifically for PostgreSQL.

The migrations package is designed to help manage database revisions in Go
applications. It works in a similar fashion to Ruby on Rails ActiveRecord
migrations:  create a directory for your migrations, add numberically ordered
SQL files with an "up" and "down" section, then write your "up" and "down"
migrations. You may then call functions in the package to migrate the database
up or down, based on the current "revision" of the database.

Version 2 of the migrations package trims down a lot of functionality that
had crept into the previous version. Note that asynchronous migrations,
Cobra/Viper integration, remote S3 migrations, and transactionless migrations
have all been removed due to lack of use. If you'd like them added, feel
free to post a ticket.

The `migrations` package is designed to be included in your product binary.
When the application deploys to the server, you can run your migrations from
the server, without manually having to setup the database. It may also be run
as a standalone tool, either from the command line or a Docker container.

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

## TODO

* Clean up test cases
* Update docs:
    * migrations.applied and migrations.rollback vs schema_migrations
    * how embedded rollbacks work
    * removal of deprecated and unused features
    * switch to postgresql only
    * cli

## Development and Testing Notes

In order to remove the PostgreSQL dependency from this package, test cases exist
in a subdirectory, `tests`, and are treated as a standalone package. This allows
us to maintain test cases, but utilize the PostgreSQL driver to test `migrations`
against a real database server.

To run these test cases:

    cd tests && make

The `Makefile` tears down any existing `migrations_test` database, creates a new
`migrations_test` database, runs the tests for migrations in a transaction,
followed by tests for migrations outside of a transaction.

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

## Using the Migrations command-line tool

The Migrations v2 includes a CLI tool to run migrations standalone, without
needing to embed them in your application. Simply install the CLI tool and
make sure the Go `bin` directory is in your path:

    $ go install github.com/sbowman/migrations/v2/cli
    $ migrate --version=12 --sql=./sql --db=postgres://localhost/myapp_db?sslmode=disable 

Use `migrate --help` for details on the available parameters.

## Rollbacks

Migrations v2 stores the rollback ("down") SQL in the database. This makes a
database migration rollback much simpler, as the migrations package doesn't
need the SQL files to rollback.

For example, you could deploy version 1.3 of your application, realize there is
a bug, then redeploy version 1.2. The migrations package can recognize the
highest version of SQL files available is lower than the migrations applied to
the database, and can run the rollback using the SQL embedded in the database
table `schema_rollbacks.`

The migrations package manages this with no additional work on the developer's
part. Migrations v2 will also upgrade any Migrations v1 projects to this
functionality, and the rollbacks SQL will not interfere with Migrations v1 if
you need to rollback.

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

