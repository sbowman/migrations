module github.com/sbowman/migrations/v2/cli

go 1.19

replace github.com/sbowman/migrations/v2 v2.0.0 => ../

require github.com/sbowman/migrations/v2 v2.0.0

require (
	github.com/inconshreveable/mousetrap v1.0.1 // indirect
	github.com/jackc/pgx/v5 v5.3.0 // indirect
	github.com/spf13/cobra v1.6.1 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
)
