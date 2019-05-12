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
	"github.com/wojnosystems/vsql"
)

// mySQL is largely a facade around Go's database/sql driver.
//
// This implementation satisfies all of the interfaces required for un-nested transactions using MySQL. I do have plans to implement a version that supports SAVEPOINTs and that will likely be a different implementation satisfying the TransactionNestedStarter
//
// The idea is to make everything an interface to allow for extension and carefully returned values from the interfaces where they make sense.
//
type mySQL struct {
	// implements SQLer
	vsql.SQLer

	// db is the database/sql.DB handle. This is composed in this object to hide the poor interface design and to make it easier to mock calls
	db *sql.DB
}
