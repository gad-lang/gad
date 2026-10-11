package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

// TestClassMethodClosure verifies that a method referring to a global (a
// closure: it captures it) registers by its name, as any method does — the
// closure keeps its function's name, named params and returns.
func TestClassMethodClosure(t *testing.T) {
	testExpectRun(t, `
		class Form { a str }
		class Start {
			count int
			methods {
				next(r) { return this.count < 10 ? Form.a : nil }
				label(; prefix="n") => prefix + this.count
			}
		}
		s := Start(; count=3)
		return [s.next(nil) == Form.a, s.label(; prefix="#")]`,
		nil, Array{True, Str("#3")})
}
