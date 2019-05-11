package vsql_mysql

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/go-sql-driver/mysql"
	_ "github.com/go-sql-driver/mysql"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
	"vsql"
	"vsql/aggregator"
	"vsql/param"
	"vsql/vquery"
	"vsql/vrow"
	"vsql/vrows"
	"vsql/vstmt"
)

// A basic test for database connectivity
func TestMySQL_Ping(t *testing.T) {
	// create a connection
	c := mustConnect(t)

	// create a table
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

func TestMySQL_InsertQuery(t *testing.T) {
	// create a connection
	c := mustConnect(t)

	// create a table
	tableNamed := mustCreateTable(t, c)
	defer mustDropTable(t, c, tableNamed)

	// insert rows
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

	queryString := "INSERT INTO " + tableNamed + " (name, age) VALUES (:name, :age)"
	for i := range data {
		q := param.NewNamedWithData(queryString,
			map[string]interface{}{
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
	err := vrow.QueryEach(c,
		context.Background(),
		param.NewAppend("SELECT name, age FROM "+tableNamed+" ORDER BY id"),
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
}

func TestTransaction_Rollback(t *testing.T) {
	// create a connection
	c := mustConnect(t)

	// create a table
	tableNamed := mustCreateTable(t, c)
	defer mustDropTable(t, c, tableNamed)

	err := vsql.Txn(c, context.Background(), nil, func(tx vsql.QueryExecer) (rollback bool, err error) {
		_, err = tx.Insert(context.Background(), param.NewAppendWithData("INSERT INTO `"+tableNamed+"` (name,age) VALUES (?,?)", "chris", 21))
		if err != nil {
			t.Error("Error not expected when inserting data")
		}

		count, err := aggregator.Count(context.Background(), tx, param.New("SELECT COUNT(*) FROM `"+tableNamed+"`"))
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

	count, err := aggregator.Count(context.Background(), c, param.New("SELECT COUNT(*) FROM `"+tableNamed+"`"))
	if err != nil {
		t.Error("Error not expected when counting data")
	}
	if 0 != count {
		t.Errorf(`Expected to rollback the insert, but inserted %d`, count)
	}
}

func TestTransaction_Commit(t *testing.T) {
	// create a connection
	c := mustConnect(t)

	// create a table
	tableNamed := mustCreateTable(t, c)
	defer mustDropTable(t, c, tableNamed)

	err := vsql.Txn(c, context.Background(), nil, func(tx vsql.QueryExecer) (rollback bool, err error) {
		_, err = tx.Insert(context.Background(), param.NewAppendWithData("INSERT INTO `"+tableNamed+"` (name,age) VALUES (?,?)", "chris", 21))
		if err != nil {
			t.Error("Error not expected when inserting data")
		}
		return
	})
	if err != nil {
		t.Fatal("error starting transaction")
	}

	count, err := aggregator.Count(context.Background(), c, param.New("SELECT COUNT(*) FROM `"+tableNamed+"`"))
	if err != nil {
		t.Error("Error not expected when counting data")
	}
	if 1 != count {
		t.Errorf(`Expected to commit the insert, but inserted %d`, count)
	}
}

func TestTransactionStatement_Commit(t *testing.T) {
	// create a connection
	c := mustConnect(t)

	// create a table
	tableNamed := mustCreateTable(t, c)
	defer mustDropTable(t, c, tableNamed)

	err := vsql.Txn(c, context.Background(), nil, func(tx vsql.QueryExecer) (rollback bool, err error) {
		var s vstmt.Statementer
		s, err = tx.Prepare(context.Background(), param.New("INSERT INTO `"+tableNamed+"` (name,age) VALUES (?,?)"))
		if err != nil {
			t.Fatal("Error not expected when preparing data")
		}

		_, err = s.Insert(context.Background(), param.NewAppendWithData("INSERT INTO `"+tableNamed+"` (name,age) VALUES (?,?)", "chris", 21))
		return
	})
	if err != nil {
		t.Fatal("error starting transaction")
	}

	count, err := aggregator.Count(context.Background(), c, param.New("SELECT COUNT(*) FROM `"+tableNamed+"`"))
	if err != nil {
		t.Error("Error not expected when counting data")
	}
	if 1 != count {
		t.Errorf(`Expected to commit the insert, but inserted %d`, count)
	}
}

func TestTransactionStatement_Rollback(t *testing.T) {
	// create a connection
	c := mustConnect(t)

	// create a table
	tableNamed := mustCreateTable(t, c)
	defer mustDropTable(t, c, tableNamed)

	err := vsql.Txn(c, context.Background(), nil, func(tx vsql.QueryExecer) (rollback bool, err error) {
		var s vstmt.Statementer
		s, err = tx.Prepare(context.Background(), param.New("INSERT INTO `"+tableNamed+"` (name,age) VALUES (?,?)"))
		if err != nil {
			t.Fatal("Error not expected when preparing data")
		}

		_, err = s.Insert(context.Background(), param.NewAppendWithData("INSERT INTO `"+tableNamed+"` (name,age) VALUES (?,?)", "chris", 21))
		return true, nil
	})
	if err != nil {
		t.Fatal("error starting transaction")
	}

	count, err := aggregator.Count(context.Background(), c, param.New("SELECT COUNT(*) FROM `"+tableNamed+"`"))
	if err != nil {
		t.Error("Error not expected when counting data")
	}
	if 0 != count {
		t.Errorf(`Expected to rollback the insert, but inserted %d`, count)
	}
}

// mustConnect creates a database connection to a MySQL server as indicated by the following OS Environment variables:
// MYSQL_USER: the username to use to connect to the MySQL database server
// MYSQL_PASSWORD: the password for MYSQL_USER to use to connect to the MySQL database server
// MYSQL_ADDR: the tcp/unix addr string for the running MySQL database server
// MYSQL_DBNAME: the database/schema to use
//
// Permissions: The MYSQL_USER you use needs to have the ability to add and remove tables
func mustConnect(t *testing.T) (s vsql.SQLer) {
	var err error
	s, err = NewMySQL(func() (db *sql.DB, err error) {
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
		return sql.Open("mysql", cfg.FormatDSN())
	})
	if err != nil {
		t.Fatal("unable to initialize the MySQL driver")
	}
	return
}

func mustCreateTable(t *testing.T, execer vquery.Execer) (tableName string) {
	tableName = fmt.Sprintf("t%d", nextId())
	_, err := execer.Exec(context.Background(), param.NewAppend("CREATE TABLE IF NOT EXISTS `"+tableName+"` ( id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY, name VARCHAR(255), age TINYINT UNSIGNED )"))
	if err != nil {
		t.Fatalf(`Unable to create table named: "%s". Err: %#v`, tableName, err)
	}
	return
}

func mustDropTable(t *testing.T, execer vquery.Execer, tableName string) {
	if len(tableName) == 0 {
		// do nothing
		return
	}
	_, err := execer.Exec(context.Background(), param.NewNamedWithData("DROP TABLE `"+tableName+"`", map[string]interface{}{"tableName": tableName}))
	if err != nil {
		t.Fatalf(`Unable to drop table named: "%s". Err: %#v`, tableName, err)
	}
}

func nextId() int64 {
	mu.Lock()
	defer mu.Unlock()
	uniqueId++
	return uniqueId
}

var uniqueId int64
var mu sync.Mutex

func init() {
	uniqueId = time.Now().Unix()
}
