module github.com/sbowman/migrations/tests

replace github.com/sbowman/migrations v1.4.0 => ../

require (
	github.com/jackc/pgx/v4 v4.13.0
	github.com/sbowman/migrations v1.4.0
)

go 1.16
