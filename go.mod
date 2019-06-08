module github.com/wojnosystems/vsql_mysql

go 1.12

require (
	github.com/go-sql-driver/mysql v1.4.1
	github.com/wojnosystems/vsql v0.0.13
	github.com/wojnosystems/vsql_engine v0.0.12
	github.com/wojnosystems/vsql_engine_go v0.0.3
	google.golang.org/appengine v1.6.0 // indirect
)

//replace github.com/wojnosystems/vsql_engine_go => ../vsql_engine_go

//replace github.com/wojnosystems/vsql_engine => ../vsql_engine
