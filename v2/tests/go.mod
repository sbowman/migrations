module github.com/sbowman/migrations/v2/tests

go 1.19

replace github.com/sbowman/migrations/v2 v2.0.0 => ../

replace github.com/sbowman/migrations v1.0.0 => ../../

require (
	github.com/jackc/pgx/v5 v5.3.0
	github.com/sbowman/migrations v1.0.0
	github.com/sbowman/migrations/v2 v2.0.0
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	golang.org/x/crypto v0.6.0 // indirect
	golang.org/x/text v0.7.0 // indirect
)
