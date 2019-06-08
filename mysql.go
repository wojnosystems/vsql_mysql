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
	"database/sql"
	"github.com/wojnosystems/vsql/interpolation_strategy"
	"github.com/wojnosystems/vsql_engine"
	"github.com/wojnosystems/vsql_engine_go"
)

// Creates a new MySQL connection manager
//
// Some questions you may have:
//  * "driverFactory? WTF?" Yes. I didn't implement the way to create the driver for you. This is a part I do not want to hide from you. You need to be in-control of this. This is how the database version has been de-coupled from this interface. As long as queries and transactions operate in the same manner, you should have no issue using this module in the future
//
// @param driverFactory is where you create your traditional Go-lang database/sql.DB object. If nil is returned for db, then nil will be returned by this method
// @return s, the SQLer object. Note, due to the way MySQL works, it does not support Nested transactions and, as such, this method does not return an object that supports that interface.
func InstallMySQL(engine vsql_engine.SingleTXer, db *sql.DB) {
	vsql_engine_go.InstallSingle(engine, db, func() interpolation_strategy.InterpolateStrategy { return &mySQLParamInterpolateStrategyDefault })
}
