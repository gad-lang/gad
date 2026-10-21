package gad_test

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gad-lang/gad"
)

// run compiles src and runs it with globals — Go values made gad values by
// gad.MustNewReflectValue —, printing what it returns, or its error.
func run(src string, globals gad.Dict) {
	st := gad.NewSymbolTable(gad.NewBuiltins().NameSet)
	for name := range globals {
		st.DefineGlobal(name)
	}
	res, err := gad.Compile(st, []byte(src), gad.CompileOptions{})
	if err != nil {
		fmt.Println("compile:", err)
		return
	}
	ret, err := gad.NewVM(gad.NewBuiltins().Build(), res.Bytecode).RunOpts(&gad.RunOpts{Globals: globals})
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(ret)
}

type Account struct {
	Owner   string
	Balance int
	Tags    []string
	// OnChange is a callback: a gad function may be assigned to it.
	OnChange func(balance int)
}

// Total is a method of the value.
func (a Account) Total(extra int) int { return a.Balance + extra }

// Deposit is a method of the pointer: it changes the account.
func (a *Account) Deposit(n int) {
	a.Balance += n
	if a.OnChange != nil {
		a.OnChange(a.Balance)
	}
}

// Withdraw fails with an error: in gad, an error a `try` catches.
func (a *Account) Withdraw(n int) (int, error) {
	if n > a.Balance {
		return a.Balance, errors.New("insufficient funds")
	}
	a.Balance -= n
	return a.Balance, nil
}

// A struct: its fields read and written, its methods called (of the value
// and of the pointer), a method as a value, an error returned by Go caught.
func Example_reflectStruct() {
	acc := &Account{Owner: "ana", Balance: 10, Tags: []string{"vip"}}
	run(`global acc
acc.Deposit(5)
acc.Owner = "Ana"
total := acc.Total
r := [acc.Owner, acc.Balance, total(100), acc.Tags[0], len(acc.Tags)]
try { acc.Withdraw(1000) } catch e { r += str(e) }
return r`, gad.Dict{"acc": gad.MustNewReflectValue(acc)})
	fmt.Println(acc.Owner, acc.Balance) // the Go value changed
	// Output:
	// ["Ana", 15, 115, "vip", 1, "error: insufficient funds"]
	// Ana 15
}

// Slices and maps: indexed, iterated, appended to; gad's values converted to
// the Go types (an int of gad to int, an array to []int, a dict to a map).
func Example_reflectSlicesAndMaps() {
	scores := map[string]int{"ana": 1}
	ids := []int{10, 20}
	sum := func(xs []int) (s int) {
		for _, x := range xs {
			s += x
		}
		return
	}
	merge := func(m map[string]int) int { return len(m) }
	run(`global (scores, ids, sum, merge)
scores["bia"] = 2
names := []
for k, v in scores { names += k + "=" + str(v) }
ids += 30                          // appended: the Go slice too (a pointer)
ids ++= [40, 50]
return [sort(names), ids[2], len(ids), sum([1, 2, 3]), merge({a: 1, b: 2})]`, gad.Dict{
		"scores": gad.MustNewReflectValue(scores),
		"ids":    gad.MustNewReflectValue(&ids),
		"sum":    gad.MustNewReflectValue(sum),
		"merge":  gad.MustNewReflectValue(merge),
	})
	fmt.Println(scores, ids)
	// Output:
	// [["ana=1", "bia=2"], 30, 5, 6, 2]
	// map[ana:1 bia:2] [10 20 30 40 50]
}

// Callbacks: a gad function given to a Go function whose parameter is a
// function type (called through the VM, synchronously); its error returned
// when the type returns one; a Go function returned by Go stays itself.
func Example_reflectCallbacks() {
	mapInts := func(xs []int, f func(int) int) []int {
		out := make([]int, len(xs))
		for i, x := range xs {
			out[i] = f(x)
		}
		return out
	}
	attempt := func(f func(string) (string, error)) string {
		s, err := f("go")
		if err != nil {
			return "failed: " + err.Error()
		}
		return s
	}
	multiplier := func(n int) func(int) int { return func(x int) int { return x * n } }
	acc := &Account{Balance: 1}
	run(`global (mapInts, attempt, multiplier, acc, log)
times3 := multiplier(3)           // a Go function, returned as it is
acc.OnChange = (b) => { log += b }
acc.Deposit(2); acc.Deposit(3)
return [
	mapInts([1, 2, 3], (x) => x * 10),
	mapInts([1, 2], times3),       // given back to Go: the same function
	attempt((s) => s + "!"),
	attempt((s) => { throw "no" }),
	log,
]`, gad.Dict{
		"mapInts":    gad.MustNewReflectValue(mapInts),
		"attempt":    gad.MustNewReflectValue(attempt),
		"multiplier": gad.MustNewReflectValue(multiplier),
		"acc":        gad.MustNewReflectValue(acc),
		"log":        gad.Array{},
	})
	// Output:
	// [[10, 20, 30], [3, 6], "go!", "failed: error: no", [3, 6]]
}

// Channels: received by `for v in ch` until closed (a Go goroutine sends),
// sent to by `ch.send(v)` (Go receives), `ch.recv()` a [value, ok] pair.
func Example_reflectChannels() {
	jobs := make(chan string)
	go func() {
		for _, j := range []string{"a", "b", "c"} {
			jobs <- j
		}
		close(jobs)
	}()
	results := make(chan int, 3)
	run(`global (jobs, results)
seen := []
for i, job in jobs {
	seen += job
	results.send(i * 10)
}
results.close()
first := results.recv()
return [seen, first, results.cap()]`, gad.Dict{
		"jobs":    gad.MustNewReflectValue(jobs),
		"results": gad.MustNewReflectValue(results),
	})
	var rest []string
	for r := range results {
		rest = append(rest, fmt.Sprint(r))
	}
	fmt.Println(strings.Join(rest, ","))
	// Output:
	// [["a", "b", "c"], [0, true], 3]
	// 10,20
}
