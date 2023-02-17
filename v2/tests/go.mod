module github.com/sbowman/migrations/tests

replace github.com/sbowman/migrations/v2 v2.0.0 => ../

require (
	github.com/jackc/pgx/v5 v5.3.0 // indirect
	github.com/sbowman/migrations/v2 v2.0.0
)

go 1.16
