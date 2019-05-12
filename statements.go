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
	"github.com/wojnosystems/vsql/param"
	"github.com/wojnosystems/vsql/vresult"
	"github.com/wojnosystems/vsql/vrows"
	"github.com/wojnosystems/vsql/vstmt"
)

// mysqlStatement is a representation of a prepared statement that is NOT running in the context of a transaction
type mysqlStatement struct {
	vstmt.Statementer
	stmt *sql.Stmt
}

// Query see github.com/wojnosystems/vsql/vstmt/statements.go#Statementer
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

// Insert see github.com/wojnosystems/vsql/vstmt/statements.go#Statementer
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

// Exec see github.com/wojnosystems/vsql/vstmt/statements.go#Statementer
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

// Close see github.com/wojnosystems/vsql/vstmt/statements.go#Statementer
func (m *mysqlStatement) Close() error {
	return m.stmt.Close()
}
