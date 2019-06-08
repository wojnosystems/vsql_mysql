# Getting started

## What is this?

A better way to use databases in Go! This is the MySQL/MariaDB implementation of an vsql_engine middleware for the vsql interface.

This allows you to inject Go's database/sql with MySQL into your application so that you can use the database in conjunction with callbacks on certain events.

## Quickstart

1. Get a database-specific implementation: ```go get github.com/wojnosystems/vsql_mysql```
1. Create your program:

```go
package mypkg

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/go-sql-driver/mysql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/wojnosystems/vsql"
	"github.com/wojnosystems/vsql/aggregator"
	"github.com/wojnosystems/vsql/vparam"
	"github.com/wojnosystems/vsql/vquery"
	"github.com/wojnosystems/vsql/vrow"
	"github.com/wojnosystems/vsql/vrows"
	"github.com/wojnosystems/vsql/vstmt"
	"github.com/wojnosystems/vsql_engine"
	"github.com/wojnosystems/vsql_engine/engine_context"
	"github.com/wojnosystems/vsql_mysql"
	"log"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// A basic test for database connectivity
func TestMySQL_Ping(t *testing.T) {
	// create a table
	c := mustConnect(t)
	defer func() { _ = c.Close() }()
	err := c.Ping(context.Background())
	if err != nil {
		t.Error("Unable to ping the MySQL database server")
	}
}

// userRecord is a test record
type userRecord struct {
	name string
	age  int
}

// Do multiple inserts and check that the values were inserted using select
func TestMySQL_InsertQuery(t *testing.T) {
	// rows to insert
	data := []userRecord{
		{
			name: "chris",
			age:  30,
		},
		{
			name: "sam",
			age:  25,
		},
		{
			name: "brian",
			age:  37,
		},
	}

	c := mustConnect(t)
	defer func() { _ = c.Close() }()
	mustTemporaryTable(t, c, func(tableName string) {
		queryString := fmt.Sprint(`INSERT INTO `, vsql.BT(tableName), ` (name, age) VALUES (:name, :age)`)
		for i := range data {
			q := vparam.NewNamedWithData(queryString,
				vsql.H{
					"name": data[i].name,
					"age":  data[i].age,
				})
			res, err := c.Insert(context.Background(), q)
			if err != nil {
				t.Fatal(err)
			}
			ra, err := res.RowsAffected()
			if err != nil {
				t.Fatal(err)
			}
			if ra != 1 {
				t.Error("expected to insert a single row")
			}
		}

		// Get the users back
		results := make([]userRecord, 0, len(data))
		queryString = fmt.Sprint(`SELECT name, age FROM `, vsql.BT(tableName), ` ORDER BY id`)
		err := vrow.QueryEach(c,
			context.Background(),
			vparam.NewAppend(queryString),
			func(r vrows.Rower) (stop bool, err error) {
				ur := userRecord{}
				err = r.Scan(&ur.name, &ur.age)
				if err == nil {
					results = append(results, ur)
				}
				return
			})
		if err != nil {
			t.Error("QueryEach should not have returned an error, but did ", err)
		}

		// Ensure that we read 3 items:
		if len(results) != len(data) {
			t.Errorf(`Expected %d results, but got %d`, len(data), len(results))
		}
		for i := range data {
			if data[i].name != results[i].name {
				t.Errorf(`Data mis-match, expected name: "%s" got "%s"`, data[i].name, results[i].name)
			}
		}
	})
}

// start a transaction, check that the value was persisted, rollback and ensure that the value is not longer visible
func TestTransaction_Rollback(t *testing.T) {
	// create a connection
	c := mustConnect(t)
	defer func() { _ = c.Close() }()

	// create a table
	mustTemporaryTable(t, c, func(tableName string) {
		err := vsql.Txn(c, context.Background(), nil, func(tx vsql.QueryExecer) (commit bool, err error) {
			_, err = tx.Insert(context.Background(), vparam.NewAppendWithData("INSERT INTO `"+tableName+"` (name,age) VALUES (?,?)", "chris", 21))
			if err != nil {
				t.Error("Error not expected when inserting data")
				return false, err
			}

			count, err := aggregator.Count(context.Background(), tx, vparam.New("SELECT COUNT(*) FROM `"+tableName+"`"))
			if err != nil {
				t.Error("Error not expected when counting data")
			}
			if 1 != count {
				t.Errorf(`Expected to insert 1 record, but inserted %d`, count)
			}
			return
		})
		if err != nil {
			t.Fatal("error starting transaction")
		}

		count, err := aggregator.Count(context.Background(), c, vparam.New("SELECT COUNT(*) FROM `"+tableName+"`"))
		if err != nil {
			t.Error("Error not expected when counting data")
		}
		if 0 != count {
			t.Errorf(`Expected to rollback the insert, but inserted %d`, count)
		}
	})
}

// start a transaction, check that the value was persisted, commit and ensure that the value is still visible
func TestTransaction_Commit(t *testing.T) {
	// create a connection
	c := mustConnect(t)
	defer func() { _ = c.Close() }()

	// create a table
	mustTemporaryTable(t, c, func(tableName string) {
		err := vsql.Txn(c, context.Background(), nil, func(tx vsql.QueryExecer) (commit bool, err error) {
			_, err = tx.Insert(context.Background(), vparam.NewAppendWithData("INSERT INTO `"+tableName+"` (name,age) VALUES (?,?)", "chris", 21))
			if err != nil {
				t.Error("Error not expected when inserting data")
			}
			return true, err
		})
		if err != nil {
			t.Fatal("error starting transaction")
		}

		count, err := aggregator.Count(context.Background(), c, vparam.New("SELECT COUNT(*) FROM `"+tableName+"`"))
		if err != nil {
			t.Error("Error not expected when counting data")
		}
		if 1 != count {
			t.Errorf(`Expected to commit the insert, but inserted %d`, count)
		}
	})
}

// start a transaction, build a prepared statement, insert a value, check that the value was persisted, commit and ensure that the value is still visible
func TestTransactionStatement_Commit(t *testing.T) {
	// create a connection
	c := mustConnect(t)
	defer func() { _ = c.Close() }()

	// create a table
	mustTemporaryTable(t, c, func(tableName string) {
		err := vsql.Txn(c, context.Background(), nil, func(tx vsql.QueryExecer) (commit bool, err error) {
			var s vstmt.Statementer
			s, err = tx.Prepare(context.Background(), vparam.New("INSERT INTO `"+tableName+"` (name,age) VALUES (?,?)"))
			if err != nil {
				t.Fatal("Error not expected when preparing data")
			}

			_, err = s.Insert(context.Background(), vparam.NewAppendWithData("INSERT INTO `"+tableName+"` (name,age) VALUES (?,?)", "chris", 21))
			return true, err
		})
		if err != nil {
			t.Fatal("error starting transaction")
		}

		count, err := aggregator.Count(context.Background(), c, vparam.New("SELECT COUNT(*) FROM `"+tableName+"`"))
		if err != nil {
			t.Error("Error not expected when counting data")
		}
		if 1 != count {
			t.Errorf(`Expected to commit the insert, but inserted %d`, count)
		}
	})
}

// start a transaction, build a prepared statement, insert a value, check that the value was persisted, rollback and ensure that the value is no longer visible
func TestTransactionStatement_Rollback(t *testing.T) {
	// create a connection
	c := mustConnect(t)
	defer func() { _ = c.Close() }()

	// create a table
	mustTemporaryTable(t, c, func(tableName string) {
		err := vsql.Txn(c, context.Background(), nil, func(tx vsql.QueryExecer) (commit bool, err error) {
			var s vstmt.Statementer
			s, err = tx.Prepare(context.Background(), vparam.New("INSERT INTO `"+tableName+"` (name,age) VALUES (?,?)"))
			if err != nil {
				t.Fatal("Error not expected when preparing data")
			}

			_, err = s.Insert(context.Background(), vparam.NewAppendWithData("INSERT INTO `"+tableName+"` (name,age) VALUES (?,?)", "chris", 21))
			return
		})
		if err != nil {
			t.Fatal("error starting transaction")
		}

		count, err := aggregator.Count(context.Background(), c, vparam.New("SELECT COUNT(*) FROM `"+tableName+"`"))
		if err != nil {
			t.Error("Error not expected when counting data")
		}
		if 0 != count {
			t.Errorf(`Expected to rollback the insert, but inserted %d`, count)
		}
	})
}

// mustConnect creates a database connection to a MySQL server as indicated by the following OS Environment variables:
// MYSQL_USER: the username to use to connect to the MySQL database server
// MYSQL_PASSWORD: the password for MYSQL_USER to use to connect to the MySQL database server
// MYSQL_ADDR: the tcp/unix addr string for the running MySQL database server
// MYSQL_DBNAME: the database/schema to use
//
// Permissions: The MYSQL_USER you use needs to have the ability to add and remove tables
func mustConnect(t *testing.T) (s vsql.SQLer) {
	cfg := mysql.Config{
		User:                 os.Getenv("MYSQL_USER"),
		Passwd:               os.Getenv("MYSQL_PASSWORD"),
		Addr:                 os.Getenv("MYSQL_ADDR"),
		DBName:               os.Getenv("MYSQL_DBNAME"),
		AllowNativePasswords: true,
		AllowOldPasswords:    true,
	}
	if strings.HasPrefix(cfg.Addr, "unix") {
		cfg.Net = "unix"
	} else {
		cfg.Net = "tcp"
	}
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatal("unable to initialize the MySQL driver", err)
		return nil
	}

	engine := vsql_engine.NewSingle()
	InstallMySQL(engine, db)
	engine.RowsNextMW().Prepend(func(ctx context.Context, c engine_context.RowsNexter) {
		value, ok := c.KeyValues().Get("mykey")
		if ok {
			log.Println(value.(string))
		} else {
			log.Println("mykey not found")
		}
		c.Next(ctx)
	})
	engine.BeginMW().Prepend(func(ctx context.Context, c engine_context.Beginner) {
		c.KeyValues().Set("transaction", "started")
		c.Next(ctx)
	})
	engine.CommitMW().Prepend(func(ctx context.Context, c engine_context.Beginner) {
		c.KeyValues().Set("transaction", "committed")
		c.Next(ctx)
	})
	engine.RollbackMW().Prepend(func(ctx context.Context, c engine_context.Beginner) {
		c.KeyValues().Set("transaction", "rolled back")
		c.Next(ctx)
	})
	return engine
}

// mustCreateTable creates a table with a "random" name (based on the current time) testing fatals are triggered if this fails
func mustCreateTable(t *testing.T, execer vquery.Execer) (tableName string) {
	tableName = fmt.Sprintf("t%d", nextId())
	_, err := execer.Exec(context.Background(), vparam.NewAppend(
		fmt.Sprint(`CREATE TABLE IF NOT EXISTS `, tableName,
			` ( id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY, name VARCHAR(255), age TINYINT UNSIGNED )`)))
	if err != nil {
		t.Fatalf(`Unable to create table named: "%s". Err: %#v`, tableName, err)
	}
	return
}

// mustDropTable deletes the table named tableName. testing fatals are triggered if this fails
func mustDropTable(t *testing.T, execer vquery.Execer, tableName string) {
	if len(tableName) == 0 {
		// do nothing
		return
	}
	_, err := execer.Exec(context.Background(), vparam.New("DROP TABLE `"+tableName+"`"))
	if err != nil {
		t.Fatalf(`Unable to drop table named: "%s". Err: %#v`, tableName, err)
	}
}

// mustTemporaryTable is a wrapper to create and "guarantee" clean up of the table used in the tests
func mustTemporaryTable(t *testing.T, execer vquery.Execer, f func(tableName string)) {
	tableName := mustCreateTable(t, execer)
	defer mustDropTable(t, execer, tableName)
	f(tableName)
}

// nextId gets the next monotonically increasing ID in the set
func nextId() int64 {
	uniqueIdMU.Lock()
	defer uniqueIdMU.Unlock()
	uniqueId++
	return uniqueId
}

var uniqueId int64
var uniqueIdMU sync.Mutex

func init() {
	uniqueId = time.Now().Unix()
}

```

# Motivation

Look, Go has a lot going for it and the packages that are part of core are largely awesome. Nobody writes perfect code and this module itself isn't perfect. Thank you to the Go team for all your hard work. Let me give back with a bit of my own for this module ;).

I started learning Go for a work project not too long ago. I've been struggling with the database interfaces that come stock with Go. While I can see why some decisions were made, the interface is cumbersome and difficult to extend. It's impossible to mock out of the box and the interfaces are non-existent. Instead of thinking through what to expose, the objects are shared directly and expose interface calls to the developer, even if they are not sensical in that sense. I've written a version of this library for work, but it's hodgepodge and also didn't consider the larger implications for interfaces (mostly because I had no idea how to use them properly at the time).

One of my biggest pet peeves was in writing a function that implemented a database request and took in a `*sql.DB`. But that means that the code assumes that it's only run OUTSIDE of a transaction. If you wanted that call to run within a transaction, you had to write the method over again or take in an optional argument to represent an optional transaction state. But this is error-prone and extremely un-clean code. The method implementing a database call should not really care if it's in a transaction or not and should take in a vquery.Queryer instead of an object that advertises details about transactions.

However, because some databases don't support nested transactions, I've split the interfaces based on the capabilities of the underlying database. One supports Nested Transactions, the other does not. The ONLY difference is that the NestedTransactionStarter returns a NestedTransactioner instead of just a Transactioner. This will help ensure that you write code to conform to the interfaces. If you decide you want to use nested transactions and move to a database that supports them, it should be as easy as changing which interfaces you use and adding the Begin calls that you need. However, the library that is implementing the vsql interfaces will need to support this.

## Contexts

I know the database/sql library has versions of the database calls that do not have contexts, I've opted to force all calls to have contexts. If you want to ignore it, use a context.Background(). This simplifies the interface and gives you more control.

# License 

Copyright 2019 Chris Wojno

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated
documentation files (the "Software"), to deal in the Software without restriction, including without limitation
the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to
permit persons to whom the Software is furnished to do so, subject to the following conditions:
The above copyright notice and this permission notice shall be included in all copies or substantial portions of the
Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE
WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS
OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR
OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

