# Go SQL Migrations

The migrations package is designed to help manage database revisions in Go
applications.  It works in a similar fashion to Ruby on Rails ActiveRecord 
migrations:  create a directory for your migrations, add numberically ordered
SQL files with an "up" and "down" section, then write your "up" and "down"
migrations.  You may then call functions in the package to migrate the database
up or down, based on the current "revision" of the database.

The "test" directory is used to test the migrations, both local and remote, and
provides a complete example of using the package.

Migrations 2.0 is completely API-compatible with Migrations 1.0.  Additionally,
all the new remote migration functionality and Cobra/Viper command support 
described below is completely optional.  It is isolated in packages so as not to 
pull in either the Cobra, Viper, or AWS packages into your application unless 
you use them.

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

## Remote S3 Migrations (2.0)

Migrations package version 2.0 adds support for "remote" migrations from the
Amazon AWS S3.  This allows you to upload your migrations to an S3 bucket, and 
have your application apply the modifications from the S3 bucket instead of a 
local disk.  This makes it easier to deploy something like AWS Lambda 
serverless compute programs.

Currently remote migrations only support AWS S3.  Additional remote storage
systems may be supported in the future.

   