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
	"github.com/wojnosystems/vsql/interpolation_strategy"
)

// MySQL driver uses ? to interpolate and not positional parameters like postgres
// This is the default strategy used by go's database/sql so there is really nothing to do here
// MySQL doesn't use positional arguments, so there is no need to store state.
// I'm creating a global strategy and using it to avoid creating an object over and over again. This is OK as there's no state.
type mySQLParamInterpolateStrategy struct {
	interpolation_strategy.InterpolateStrategy
}

// InsertPlaceholderIntoSQL injects a question mark (?) whenever a variable is intended to be substituted into the SQL call via parametrization
func (m *mySQLParamInterpolateStrategy) InsertPlaceholderIntoSQL() string {
	return "?"
}

// mySQLParamInterpolateStrategyDefault is the default parameter strategy
var mySQLParamInterpolateStrategyDefault = mySQLParamInterpolateStrategy{}
var mySQLParamInterpolateStrategyFactoryDefault = func() interpolation_strategy.InterpolateStrategy {
	return &mySQLParamInterpolateStrategyDefault
}
