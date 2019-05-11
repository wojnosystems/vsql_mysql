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
	"vsql"
	"vsql/param"
	"vsql/vresult"
	"vsql/vrows"
	"vsql/vstmt"
	"vsql/vtxn"
)

// mySQL is largely a facade around Go's database/sql
// The idea is to make everything an interface to allow for extension and carefully returned values from the interfaces where they make sense.
type mySQL struct {
	db *sql.DB
}

// Creates a new MySQL connection manager
func NewMySQL(driverFactory func() (db *sql.DB, err error)) (s vsql.SQLer, err error) {
	m := &mySQL{}
	m.db, err = driverFactory()
	return m, err
}

func (m *mySQL) Begin(ctx context.Context, txOp vtxn.TxOptioner) (n vsql.QueryExecTransactioner, err error) {
	var txo *sql.TxOptions
	if txOp != nil {
		txo = txOp.ToTxOptions()
	}
	t := &mySQLtx{}
	t.tx, err = m.db.BeginTx(ctx, txo)
	return t, nil
}

func (m *mySQL) Ping(ctx context.Context) error {
	return m.db.PingContext(ctx)
}

var mySQLParamInterpolateStrategyDefault = mySQLParamInterpolateStrategy{}

func (m *mySQL) Query(ctx context.Context, query param.Queryer) (rRows vrows.Rowser, err error) {
	var q string
	var ps []interface{}
	r := &vrows.RowsImpl{}
	q, ps, err = query.Interpolate(&mySQLParamInterpolateStrategyDefault)
	if err != nil {
		return r, err
	}
	r.SqlRows, err = m.db.QueryContext(ctx, q, ps...)
	return r, err
}

func (m *mySQL) Insert(ctx context.Context, query param.Queryer) (res vresult.InsertResulter, err error) {
	var q string
	var ps []interface{}
	sqlRes := &vresult.QueryResult{}
	q, ps, err = query.Interpolate(&mySQLParamInterpolateStrategyDefault)
	if err != nil {
		return sqlRes, err
	}
	sqlRes.SqlRes, err = m.db.ExecContext(ctx, q, ps...)
	return sqlRes, err
}

func (m *mySQL) Exec(ctx context.Context, query param.Queryer) (res vresult.Resulter, err error) {
	var q string
	var ps []interface{}
	sqlRes := &vresult.QueryResult{}
	q, ps, err = query.Interpolate(&mySQLParamInterpolateStrategyDefault)
	if err != nil {
		return sqlRes, err
	}
	sqlRes.SqlRes, err = m.db.ExecContext(ctx, q, ps...)
	return sqlRes, err
}

func (m *mySQL) Prepare(ctx context.Context, query param.Queryer) (stmtr vstmt.Statementer, err error) {
	q := query.SQLQuery(&mySQLParamInterpolateStrategyDefault)
	mStmt := &mysqlStatement{}
	mStmt.stmt, err = m.db.PrepareContext(ctx, q)
	return mStmt, err
}

type mySQLtx struct {
	tx *sql.Tx
}

func (m *mySQLtx) Commit() error {
	return m.tx.Commit()
}

func (m *mySQLtx) Rollback() error {
	return m.tx.Rollback()
}
func (m *mySQLtx) Query(ctx context.Context, query param.Queryer) (rRows vrows.Rowser, err error) {
	var q string
	var ps []interface{}
	r := &vrows.RowsImpl{}
	q, ps, err = query.Interpolate(&mySQLParamInterpolateStrategyDefault)
	if err != nil {
		return r, err
	}
	r.SqlRows, err = m.tx.QueryContext(ctx, q, ps...)
	return r, err
}

func (m *mySQLtx) Prepare(ctx context.Context, query param.Queryer) (stmtr vstmt.Statementer, err error) {
	q := query.SQLQuery(&mySQLParamInterpolateStrategyDefault)
	mStmt := &mysqlStatementTx{
		tx: m.tx,
	}
	mStmt.stmt, err = m.tx.PrepareContext(ctx, q)
	return mStmt, err
}

func (m *mySQLtx) Insert(ctx context.Context, query param.Queryer) (res vresult.InsertResulter, err error) {
	var q string
	var ps []interface{}
	sqlRes := &vresult.QueryResult{}
	q, ps, err = query.Interpolate(&mySQLParamInterpolateStrategyDefault)
	if err != nil {
		return sqlRes, err
	}
	sqlRes.SqlRes, err = m.tx.ExecContext(ctx, q, ps...)
	return sqlRes, err
}

func (m *mySQLtx) Exec(ctx context.Context, query param.Queryer) (res vresult.Resulter, err error) {
	var q string
	var ps []interface{}
	sqlRes := &vresult.QueryResult{}
	q, ps, err = query.Interpolate(&mySQLParamInterpolateStrategyDefault)
	if err != nil {
		return sqlRes, err
	}
	sqlRes.SqlRes, err = m.tx.ExecContext(ctx, q, ps...)
	return sqlRes, err
}

// MySQL driver uses ? to interpolate and not positional parameters like postgres
type mySQLParamInterpolateStrategy struct {
}

func (m *mySQLParamInterpolateStrategy) InsertPlaceholderIntoSQL() string {
	return "?"
}

type mysqlStatement struct {
	stmt *sql.Stmt
}

func (m *mysqlStatement) Query(ctx context.Context, query param.Parameterer) (rRows vrows.Rowser, err error) {
	var ps []interface{}
	sqlRes := &vrows.RowsImpl{}
	_, ps, err = query.Interpolate(&mySQLParamInterpolateStrategyDefault)
	if err != nil {
		return sqlRes, err
	}
	sqlRes.SqlRows, err = m.stmt.QueryContext(ctx, ps...)
	return sqlRes, err
}
func (m *mysqlStatement) Insert(ctx context.Context, query param.Parameterer) (res vresult.InsertResulter, err error) {
	var ps []interface{}
	sqlRes := &vresult.QueryResult{}
	_, ps, err = query.Interpolate(&mySQLParamInterpolateStrategyDefault)
	if err != nil {
		return sqlRes, err
	}
	sqlRes.SqlRes, err = m.stmt.ExecContext(ctx, ps...)
	return sqlRes, err
}
func (m *mysqlStatement) Exec(ctx context.Context, query param.Parameterer) (res vresult.Resulter, err error) {
	var ps []interface{}
	sqlRes := &vresult.QueryResult{}
	_, ps, err = query.Interpolate(&mySQLParamInterpolateStrategyDefault)
	if err != nil {
		return sqlRes, err
	}
	sqlRes.SqlRes, err = m.stmt.ExecContext(ctx, ps...)
	return sqlRes, err
}
func (m *mysqlStatement) Close() error {
	return m.stmt.Close()
}

type mysqlStatementTx struct {
	stmt *sql.Stmt
	tx   *sql.Tx
}

func (m *mysqlStatementTx) Query(ctx context.Context, query param.Parameterer) (rRows vrows.Rowser, err error) {
	var ps []interface{}
	sqlRes := &vrows.RowsImpl{}
	_, ps, err = query.Interpolate(&mySQLParamInterpolateStrategyDefault)
	if err != nil {
		return sqlRes, err
	}
	sqlRes.SqlRows, err = m.tx.StmtContext(ctx, m.stmt).QueryContext(ctx, ps...)
	return sqlRes, err
}
func (m *mysqlStatementTx) Insert(ctx context.Context, query param.Parameterer) (res vresult.InsertResulter, err error) {
	var ps []interface{}
	sqlRes := &vresult.QueryResult{}
	_, ps, err = query.Interpolate(&mySQLParamInterpolateStrategyDefault)
	if err != nil {
		return sqlRes, err
	}
	sqlRes.SqlRes, err = m.tx.StmtContext(ctx, m.stmt).ExecContext(ctx, ps...)
	return sqlRes, err
}
func (m *mysqlStatementTx) Exec(ctx context.Context, query param.Parameterer) (res vresult.Resulter, err error) {
	var ps []interface{}
	sqlRes := &vresult.QueryResult{}
	_, ps, err = query.Interpolate(&mySQLParamInterpolateStrategyDefault)
	if err != nil {
		return sqlRes, err
	}
	sqlRes.SqlRes, err = m.tx.StmtContext(ctx, m.stmt).ExecContext(ctx, ps...)
	return sqlRes, err
}
func (m *mysqlStatementTx) Close() error {
	return m.stmt.Close()
}
