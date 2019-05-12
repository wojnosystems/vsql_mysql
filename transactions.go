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
	"github.com/wojnosystems/vsql"
	"github.com/wojnosystems/vsql/param"
	"github.com/wojnosystems/vsql/vresult"
	"github.com/wojnosystems/vsql/vrows"
	"github.com/wojnosystems/vsql/vstmt"
)

// mySQLtx is the internal model for transactions
type mySQLtx struct {
	vsql.QueryExecTransactioner
	tx *sql.Tx
}

// Commit see github.com/wojnosystems/vsql/transactions.go#Transactioner
func (m *mySQLtx) Commit() error {
	return m.tx.Commit()
}

// Rollback see github.com/wojnosystems/vsql/transactions.go#Transactioner
func (m *mySQLtx) Rollback() error {
	return m.tx.Rollback()
}

// Query see github.com/wojnosystems/vsql/interfaces.go#QueryExecer
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

// Insert see github.com/wojnosystems/vsql/interfaces.go#QueryExecer
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

// Exec see github.com/wojnosystems/vsql/interfaces.go#QueryExecer
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

// Prepare see github.com/wojnosystems/vsql/vstmt/statements.go#Preparer
func (m *mySQLtx) Prepare(ctx context.Context, query param.Queryer) (stmtr vstmt.Statementer, err error) {
	q := query.SQLQuery(&mySQLParamInterpolateStrategyDefault)
	mStmt := &mysqlStatementTx{
		tx: m.tx,
	}
	mStmt.stmt, err = m.tx.PrepareContext(ctx, q)
	return mStmt, err
}

// mysqlStatementTx is a prepared statement within the context of a transaction
// sadly, based on the way the database/sql package works, most of this code has to be duplicated with the non-transaction version.
// however, this should save people implementing this code downstream much typing and testing. The amount of code written here is also fairly small due to the way the vsql package is composed so few changes, if any, will be required.
type mysqlStatementTx struct {
	vstmt.Statementer
	stmt *sql.Stmt
	tx   *sql.Tx
}

// Query see github.com/wojnosystems/vsql/vstmt/statements.go#Statementer
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

// Insert see github.com/wojnosystems/vsql/vstmt/statements.go#Statementer
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

// Exec see github.com/wojnosystems/vsql/vstmt/statements.go#Statementer
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

// Close see github.com/wojnosystems/vsql/vstmt/statements.go#Statementer
func (m *mysqlStatementTx) Close() error {
	return m.stmt.Close()
}
