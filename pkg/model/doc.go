// Package model parses JSON or YAML into a struct, converts values that fit exactly, and runs
// the rules in its validate tags. It returns the value the input meant, or an error.
//
// # Usage
//
//	type User struct {
//	    ID    int    `json:"id" validate:"required,min=1"`
//	    Name  string `json:"name" validate:"required"`
//	    Email string `json:"email" validate:"email"`
//	}
//
//	user, err := model.ParseInto[User](data)
//
// # Type Coercion
//
// Converts values when nothing is lost: "42" to 42, "true" to true, Unix seconds or RFC 3339 to
// time.Time. Out-of-range and fractional values are errors.
//
// # Validation
//
// Built-in rules: required, min, max, length (or len), email, alpha, alphanum. A tag that cannot
// run is an error on every parse. Register more with RegisterGlobalFunc.
//
// # Caching
//
// CachedParser skips the work for input it has seen before, and returns independent copies.
//
//	parser := model.NewCachedParser[User](nil)
//	defer parser.Close()
//	user, err := parser.Parse(data)
//
// Documentation: https://1mb-dev.github.io/gopantic/
package model
