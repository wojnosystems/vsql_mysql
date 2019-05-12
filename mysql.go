//Copyright 2019 Chris Wojno
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated
// documentation files (the "Software"), to deal in the Software without restriction, including without limitation
// the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to
// permit persons to whom the Software is furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all copies or substantial portions of the
// Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE
// WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS
// OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR
// OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package vsql_mysql

import (
	"context"
	"database/sql"
	"github.com/wojnosystems/vsql"
	"github.com/wojnosystems/vsql/param"
	"github.com/wojnosystems/vsql/vresult"
	"github.com/wojnosystems/vsql/vrows"
	"github.com/wojnosystems/vsql/vstmt"
	"github.com/wojnosystems/vsql/vtxn"
)

// Creates a new MySQL connection manager
//
// Some questions you may have:
//  * "driverFactory? WTF?" Yes. I didn't implement the way to create the driver for you. This is a part I do not want to hide from you. You need to be in-control of this. This is how the database version has been de-coupled from this interface. As long as queries and transactions operate in the same manner, you should have no issue using this module in the future
//
// @param driverFactory is where you create your traditional Go-lang database/sql.DB object. If nil is returned for db, then nil will be returned by this method
// @return s, the SQLer object. Note, due to the way MySQL works, it does not support Nested transactions and, as such, this method does not return an object that supports that interface.
func NewMySQL(driverFactory func() (db *sql.DB)) (s vsql.SQLer) {
	m := &mySQL{}
	m.db = driverFactory()
	if m.db == nil {
		return nil
	}
	return m
}

// Begin see github.com/wojnosystems/vsql/transactions.go#TransactionStarter
func (m *mySQL) Begin(ctx context.Context, txOp vtxn.TxOptioner) (n vsql.QueryExecTransactioner, err error) {
	var txo *sql.TxOptions
	if txOp != nil {
		txo = txOp.ToTxOptions()
	}
	t := &mySQLtx{}
	t.tx, err = m.db.BeginTx(ctx, txo)
	return t, nil
}

// Ping see github.com/wojnosystems/vsql/pinger/pinger.go#Pinger
func (m *mySQL) Ping(ctx context.Context) error {
	return m.db.PingContext(ctx)
}

// Query see github.com/wojnosystems/vsql/vquery/query.go#Queryer
func (m *mySQL) Query(ctx context.Context, query param.Queryer) (rRows vrows.Rowser, err error) {
	q, ps, err := query.Interpolate(&mySQLParamInterpolateStrategyDefault)
	if err != nil {
		return nil, err
	}
	r := &vrows.RowsImpl{}
	r.SqlRows, err = m.db.QueryContext(ctx, q, ps...)
	return r, err
}

// Insert see github.com/wojnosystems/vsql/vquery/query.go#Inserter
func (m *mySQL) Insert(ctx context.Context, query param.Queryer) (res vresult.InsertResulter, err error) {
	q, ps, err := query.Interpolate(&mySQLParamInterpolateStrategyDefault)
	if err != nil {
		return nil, err
	}
	sqlRes := &vresult.QueryResult{}
	sqlRes.SqlRes, err = m.db.ExecContext(ctx, q, ps...)
	return sqlRes, err
}

// Exec see github.com/wojnosystems/vsql/vquery/query.go#Execer
func (m *mySQL) Exec(ctx context.Context, query param.Queryer) (res vresult.Resulter, err error) {
	q, ps, err := query.Interpolate(&mySQLParamInterpolateStrategyDefault)
	if err != nil {
		return nil, err
	}
	sqlRes := &vresult.QueryResult{}
	sqlRes.SqlRes, err = m.db.ExecContext(ctx, q, ps...)
	return sqlRes, err
}

// Prepare see github.com/wojnosystems/vsql/vstmt/statement.go#Preparer
func (m *mySQL) Prepare(ctx context.Context, query param.Queryer) (stmtr vstmt.Statementer, err error) {
	q := query.SQLQuery(&mySQLParamInterpolateStrategyDefault)
	mStmt := &mysqlStatement{}
	mStmt.stmt, err = m.db.PrepareContext(ctx, q)
	return mStmt, err
}
