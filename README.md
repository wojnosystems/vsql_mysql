# Getting started

## What is this?

A better way to use databases in Go!

## Quickstart

1. Get a database-specific implementation: ```go get github.com/wojnosystems/vsql_mysql```
1. Create your program:

```go
package main

import (
	"context"
	"database/sql"
	"github.com/go-sql-driver/mysql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/wojnosystems/vsql"
	"github.com/wojnosystems/vsql/param"
	"github.com/wojnosystems/vsql/vquery"
	"github.com/wojnosystems/vsql/vrow"
	"github.com/wojnosystems/vsql/vrows"
	"github.com/wojnosystems/vsql_mysql"
	"log"
	"os"
	"strings"
)

// mustCreateTable demonstrates how to create a table
func mustCreateTable(ctx context.Context, db vquery.Execer) {
	// create your tables
	_, err := db.Exec(ctx, param.New(
		`CREATE TABLE IF NOT EXISTS mytable ( id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY, name VARCHAR(255), age TINYINT UNSIGNED )`))
	if err != nil {
		log.Fatal("cannot create table because ", err)
	}
}

// insertChris INSERT add data
func insertChris(ctx context.Context, db vquery.Inserter) {
	// Use parameterized values for safety!
	res, err := db.Insert(ctx, param.NewNamedWithData(
		// regular query, with named parameters
		`INSERT INTO mytable (name, age) VALUES (:name, :age)`,
		// This is your named placeholder map: vsql.H is just a shortcut for map[string]interface{}
		vsql.H{
			"name": "chris",
			"age":  30,
		}))
	if err != nil {
		log.Fatal("cannot insert record because ", err)
	}
	// Look up the last ID
	lastId, _ := res.LastInsertId()
	_ = lastId // do something with the last inserted id

	howManyDidIInsert, _ := res.RowsAffected()
	_ = howManyDidIInsert // do something with how many inserted
}

// getChris SELECT get data
func getChris(ctx context.Context, db vquery.Queryer) (name string, age int) {
	// Query your row. Grab the first record.
	// vrow.QueryOne is a convenience method to grab the first (and perhaps only) row returned
	// QueryOne each also handles cleaning up after the row and results
	ok, err := vrow.QueryOne(db, ctx, param.NewNamedWithData(
		// regular query, with named parameters
		`SELECT name, age FROM mytable WHERE name = :name`,
		// replacing :name with the parameterized placeholder for chris
		vsql.H{"name": "chris",}),
		// Do something with that row, scan the values or something
		func(ro vrows.Rower) (err error) {
			return ro.Scan(&name, &age)
		})
	// ok means a record was found and the Scan was completed.
	if !ok {
		log.Fatal("cannot look up record because ", err)
	}
	return
}

// getAllAges Look up lots of data
// In this query, we'll look up multiple values and return them as an array of ages
func getAllAges(ctx context.Context, db vquery.Queryer) (ages []int) {
	ages = make([]int, 0, 10)
	// vrow.QueryEach is a convenience method that iterates through results for you
	// you can tell it to STOP at any time (set it to true) or by returning any errors encountered
	// that way you only need to deal with errors at the top-level. If you pass an error, iteration will stop
	// QueryEach each also handles cleaning up after the row and results
	err := vrow.QueryEach(db, ctx,
		param.NewNamedWithData(
			`SELECT name, age FROM mytable WHERE name = :name LIMIT 1000`,
			vsql.H{"name": "chris",}),
		// Do something with each row. This is the code that extracts data from the query
		func(r vrows.Rower) (stop bool, err error) {
			var theAge int
			err = r.Scan(&theAge)
			if err == nil {
				// only append if the age was read
				ages = append(ages, theAge)
			}
			// default of stop = false, which means iteration will continue, return false to end looping
			// the err is also set from r.Scan, which means if there is an error, looping will stop
			// and that error will be returned from vrow.QueryEach
			return
		})
	if err != nil {
		log.Fatal("cannot get ages because ", err)
	}
	return
}

func main() {
	ctx := context.Background()
	// Connect to your database
	db := mustConnect()

	// Check if your database is alive
	err := db.Ping(ctx)
	if err != nil {
		log.Fatal("cannot connect to the database because ", err)
	}

	mustCreateTable(ctx, db)
	insertChris(ctx, db)
	name, age := getChris(ctx, db)
	_ = name // Do things with them
	_ = age  // Do things with them

	ages := getAllAges(ctx, db)
	_ = ages // Do things with them
}

// mustConnect connects to the database and panics if this fails.
// This method is to hide the implementation details of connecting
// for this example, but it's a good pattern to use, too
func mustConnect() (s vsql.SQLer) {
	var err error
	s = vsql_mysql.NewMySQL(func() (db *sql.DB) {
		// do your MySQL config
		cfg := mysql.Config{
			User:                 os.Getenv("MYSQL_USER"),
			Passwd:               os.Getenv("MYSQL_PASSWORD"),
			Addr:                 os.Getenv("MYSQL_ADDR"),
			DBName:               os.Getenv("MYSQL_DBNAME"),
			AllowNativePasswords: true,
		}
		if strings.HasPrefix(cfg.Addr, "unix") {
			cfg.Net = "unix"
		} else {
			cfg.Net = "tcp"
		}
		db, err = sql.Open("mysql", cfg.FormatDSN())
		return db
	})
	if err != nil {
		log.Fatal(err)
	}
	return s
}
```

# Motivation

Look, Go has a lot going for it and the packages that are part of core are largely awesome. Nobody writes perfect code and this module itself isn't perfect. Thank you to the Go team for all your hard work. Let me give back with a bit of my own for this module ;).

I started learning Go for a work project not too long ago. I've been struggling with the database interfaces that come stock with Go. While I can see why some decisions were made, the interface is cumbersome and difficult to extend. It's impossible to mock out of the box and the interfaces are non-existent. Instead of thinking through what to expose, the objects are shared directly and expose interface calls to the developer, even if they are not sensical in that sense. I've written a version of this library for work, but it's hodgepodge and also didn't consider the larger implications for interfaces (mostly because I had no idea how to use them properly at the time).

One of my biggest pet peeves was in writing a function that implemented a database request and took in a `*sql.DB`. But that means that the code assumes that it's only run OUTSIDE of a transaction. If you wanted that call to run within a transaction, you had to write the method over again or take in an optional argument to represent an optional transaction state. But this is error-prone and extremely un-clean code. The method implementing a database call should not really care if it's in a transaction or not and should take in a vquery.Queryer instead of an object that advertises details about transactions.

However, because some databases (cough!--MYSQL--cough!) don't support nested transactions, I've split the interfaces based on the capabilities of the underlying database. One supports Nested Transactions, the other does not. The ONLY difference is that the NestedTransactionStarter returns a NestedTransactioner instead of just a Transactioner. This will help ensure that you write code to conform to the interfaces. If you decide you want to use nested transactions and move to a database that supports them, it should be as easy as changing which interfaces you use and adding the Begin calls that you need. However, the library that is implementing the vsql interfaces will need to support this.

## Contexts

I know the database/sql library has versions of the database calls that do not have contexts, I've opted to force all calls to have contexts. If you want to ignore it, use a context.Background(). This simplifies the interface and gives you more control.

# Mocks

I've added stretchr/mock structs for your convenience. I'm constantly mocking database interactions and you can use these mocks to avoid having to write them yourself.

All of the mock'ed structs are named after the interface that they implement+"Mock" so if you want to mock the Pinger interface, PingerMock is what you want to add your method mocks onto.

The _test's have examples of how to use these mocks as this library is unable to connect to any actual databases.

# Other cool features

## aggregators

The is just one, but I'll likely add more aggregators. These simply extract values from queries.

aggregator.Count extracts the only value from a `SELECT COUNT(*) FROM ...` query, as this is a very common pattern

## Named parameters

sqlx got named parameters down pat. I liked it and implemented it in this library as well. I didn't look how they did it, but rolled my own. The major issue with their parameters is that they're not interfaces, which means they're not really portable and extensible :(. You can use mine like:

```go
p := param.NewNamed("SELECT * FROM users where name = :key AND magicNumber = :someNum")
p.Set("key", "value")
p.Set("someNum", 42)
``` 

The `key` and `someNum` parameters will be replaced with `value` and `42`, respectively.

You can also use the more compact version:

```go
p := param.NewNamedWithData("SELECT * FROM users where name = :key AND magicNumber = :someNum",
	vsql.H{"key": "value", "someNum": 42}
```

If you have no paramters and just want to do a blanket statement such as: `SELECT COUNT(*) FROM users`, you can use `param.New("SELECT COUNT(*) FROM users")` instead

## Row Iterators

One patterns you'll encounter over... and over... and over....... and over again is iterating over datasets in sql. With the base database/sql library, you are constantly stopping for errors along the way.

While this interface doesn't stop you from iterating over the results one at a time just as you would with database/sql, it does offer a more convenient way to write/pass queries and then iterate over the results. You just pass in a predicate function and transform your data as desired. This means you can use and re-use queries as well as use and re-use predicates to transform data. This is a tremendous de-coupling of the logic that's usually entangled in the calls, unless you wrote your own way of doing this in your implementations. I'm here to tell you to stop doing that. Stop it. Use this library instead!

### vrows.Each, vrows.QueryEach, vrows.One, & vrows.QueryOne

These are a set of helper methods to remove boiler plate from database code. Each of these methods operate on result sets. Once these methods return, they clean up the result set for the caller so that you no longer have to call Close on the vrows.Rowser object yourself.

These also remove the `sql.ErrNoRows` errors as I believe these are a mistake. No rows is not an error and usually indicates valuable information, such as the information that nothing exists. Seriously, why would you return an error for non-error states? Anyway, this library fixes that stupidity. If you use these methods, this error will be hidden and this state is communicated to you in other, non-error ways.

#### vrows.Each

Each operates on the result set of a previous query. This is the foundation for QueryEach which you'll probably end up using more as it hides the error handling and other issues so you can focus on getting stuff done.

Each is passed a vrows.Rowser, which is a list of rows expected to have been derived from a previous Query. eachRow is the function that operates on each iteration of the rows in the vrows.Rowser result set.

Here, you can do anything you want on a single row. See vrows.Rower for the interface of what you can do with it, including getting a list of column names, and scanning values out to whatever you wish.

eachRow will be called on each record/row returned from the database unless you set `stop` to true. This tells the underlying loop to stop after the current record. If you pass an error to `err`, this will also halt iteration after this record.

Within the eachRow method you pass in, you should Unmarshal your data from the database into a format you need.

#### vrows.One

One operates just like Each, but only grabs the first row. If any additional rows are present, they are ignored. One closes the record/row before it returns. If you need all of the information, consider using QueryEach/Each instead.

Notice in `theRow` that there is no `stop` variable. That's becuase there is no iteration.

The value `ok` is returned true when at least 1 record was returned by the query. `ok` can be true, even if there was a Scan error of you returned an error. 

#### vrows.QueryEach

QueryEach is just like Each, but handles the entire Query pipeline for you. This consolidates errors and hides the ErrNoRows error.

You should prefer using this method before using Each as this will hide most of the boiler plate and error repetition for you.

#### vrows.QueryOne

QueryOne is just like One, but handles the entire Query pipeline for you. This consolidates errors and hides the ErrNoRows error.

You should prefer using this method before using One as this will hide most of the boiler plate and error repetition for you.

## Backticking

Use the backtick helper:
```go
original := "mytable"
backTicked := vsql.BT(original) // => "`mytable`"
```

## Transaction Wrapping

Transactions require that you commit or roll them back after executing a set of statements. The `vsql.Txn` and `vsql.TxnNested` provides a clean way to perform transactions as a set. If you need to have transactions span to callers, then don't use these. If, however, you are composing downward dependent transactions (as in, you start a transaction and then perform queries/execs on that transaction or child transactions), this this is what you want.

These methods take care of Begin'ing the transaction and cleaning up after them using Commit/Rollback. In the event of a panic, Rollback is called. It can be used like this:

```go
err := vsql.Txn(c, context.Background(), nil, func(tx vsql.QueryExecer) (rollback bool, err error) {
    _, err = tx.Insert(context.Background(), param.NewAppendWithData("INSERT INTO `"+tableName+"` (name,age) VALUES (?,?)", "chris", 21))
    if err != nil {
        t.Error("Error not expected when inserting data")
    }

    count, err := aggregator.Count(context.Background(), tx, param.New("SELECT COUNT(*) FROM `"+tableName+"`"))
    if err != nil {
        t.Error("Error not expected when counting data")
    }
    if 1 != count {
        t.Errorf(`Expected to insert 1 record, but inserted %d`, count)
    }

    rollback = true
    return
})
if err != nil {
    t.Fatal("error starting transaction")
}
```

This begins a transaction, inserts the row, counts the row, then rolls the transaction back. If you wanted to commit the transaction, the function needs to `return true, nil`. Because return is called, this means the insert is undone. The rollback occurs when commit is not actively set to true. By default, commit is false, so if you do nothing and have the transaction return, it will be rolled back.

If you return an error, a rollback will be issued as well, but the error will be propagated up and returned to the caller of vsql.Txn.

If your code emits a panic, your transaction will be rolled back, and the panic will be re-panic'ed after the rollback

# License 

Copyright 2019 Chris Wojno

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

