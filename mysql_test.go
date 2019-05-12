//Copyright 2019 Chris Wojno
//
//Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:
//
//The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.
//
//THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package vsql_mysql

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/go-sql-driver/mysql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/wojnosystems/vsql"
	"github.com/wojnosystems/vsql/aggregator"
	"github.com/wojnosystems/vsql/param"
	"github.com/wojnosystems/vsql/vquery"
	"github.com/wojnosystems/vsql/vrow"
	"github.com/wojnosystems/vsql/vrows"
	"github.com/wojnosystems/vsql/vstmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// A basic test for database connectivity
func TestMySQL_Ping(t *testing.T) {
	// create a table
	err := mustConnect(t).Ping(context.Background())
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
	mustTemporaryTable(t, c, func(tableName string) {
		queryString := fmt.Sprint(`INSERT INTO `, vsql.BT(tableName), ` (name, age) VALUES (:name, :age)`)
		for i := range data {
			q := param.NewNamedWithData(queryString,
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
			param.NewAppend(queryString),
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

	// create a table
	mustTemporaryTable(t, c, func(tableName string) {
		err := vsql.Txn(c, context.Background(), nil, func(tx vsql.QueryExecer) (commit bool, err error) {
			_, err = tx.Insert(context.Background(), param.NewAppendWithData("INSERT INTO `"+tableName+"` (name,age) VALUES (?,?)", "chris", 21))
			if err != nil {
				t.Error("Error not expected when inserting data")
				return false, err
			}

			count, err := aggregator.Count(context.Background(), tx, param.New("SELECT COUNT(*) FROM `"+tableName+"`"))
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

		count, err := aggregator.Count(context.Background(), c, param.New("SELECT COUNT(*) FROM `"+tableName+"`"))
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

	// create a table
	mustTemporaryTable(t, c, func(tableName string) {
		err := vsql.Txn(c, context.Background(), nil, func(tx vsql.QueryExecer) (commit bool, err error) {
			_, err = tx.Insert(context.Background(), param.NewAppendWithData("INSERT INTO `"+tableName+"` (name,age) VALUES (?,?)", "chris", 21))
			if err != nil {
				t.Error("Error not expected when inserting data")
			}
			return true, err
		})
		if err != nil {
			t.Fatal("error starting transaction")
		}

		count, err := aggregator.Count(context.Background(), c, param.New("SELECT COUNT(*) FROM `"+tableName+"`"))
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

	// create a table
	mustTemporaryTable(t, c, func(tableName string) {
		err := vsql.Txn(c, context.Background(), nil, func(tx vsql.QueryExecer) (commit bool, err error) {
			var s vstmt.Statementer
			s, err = tx.Prepare(context.Background(), param.New("INSERT INTO `"+tableName+"` (name,age) VALUES (?,?)"))
			if err != nil {
				t.Fatal("Error not expected when preparing data")
			}

			_, err = s.Insert(context.Background(), param.NewAppendWithData("INSERT INTO `"+tableName+"` (name,age) VALUES (?,?)", "chris", 21))
			return true, err
		})
		if err != nil {
			t.Fatal("error starting transaction")
		}

		count, err := aggregator.Count(context.Background(), c, param.New("SELECT COUNT(*) FROM `"+tableName+"`"))
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

	// create a table
	mustTemporaryTable(t, c, func(tableName string) {
		err := vsql.Txn(c, context.Background(), nil, func(tx vsql.QueryExecer) (commit bool, err error) {
			var s vstmt.Statementer
			s, err = tx.Prepare(context.Background(), param.New("INSERT INTO `"+tableName+"` (name,age) VALUES (?,?)"))
			if err != nil {
				t.Fatal("Error not expected when preparing data")
			}

			_, err = s.Insert(context.Background(), param.NewAppendWithData("INSERT INTO `"+tableName+"` (name,age) VALUES (?,?)", "chris", 21))
			return
		})
		if err != nil {
			t.Fatal("error starting transaction")
		}

		count, err := aggregator.Count(context.Background(), c, param.New("SELECT COUNT(*) FROM `"+tableName+"`"))
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
	s = NewMySQL(func() (db *sql.DB) {
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
		return
	})
	return
}

// mustCreateTable creates a table with a "random" name (based on the current time) testing fatals are triggered if this fails
func mustCreateTable(t *testing.T, execer vquery.Execer) (tableName string) {
	tableName = fmt.Sprintf("t%d", nextId())
	_, err := execer.Exec(context.Background(), param.NewAppend(
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
	_, err := execer.Exec(context.Background(), param.NewNamedWithData("DROP TABLE `"+tableName+"`", vsql.H{"tableName": tableName}))
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
