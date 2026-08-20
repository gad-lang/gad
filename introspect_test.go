package gad_test

import (
	"context"
	"testing"

	gad "github.com/gad-lang/gad"
	"github.com/stretchr/testify/require"
)

func eval(t *testing.T, src string) gad.Object {
	t.Helper()
	e := gad.NewEval(nil, nil, gad.CompileOptions{})
	ret, _, err := e.RunScript(context.Background(), []byte(src))
	require.NoError(t, err, src)
	return ret
}

func memberKinds(ms []gad.Member) map[string]string {
	m := map[string]string{}
	for _, x := range ms {
		m[x.Name] = x.Kind
	}
	return m
}

func TestMembersDict(t *testing.T) {
	got := memberKinds(gad.Members(eval(t, `return {host: "x", port: 8080, nested: {a: 1}}`)))
	require.Equal(t, map[string]string{"host": "key", "port": "key", "nested": "key"}, got)
}

func TestMembersClassInstance(t *testing.T) {
	src := `
class Animal {
	name str = "?"
	methods {
		speak() => this.name
	}
	props {
		label = this.name
	}
}
return Animal(; name="Rex")`
	got := memberKinds(gad.Members(eval(t, src)))
	require.Equal(t, "field", got["name"])
	require.Equal(t, "method", got["speak"])
	require.Equal(t, "property", got["label"])
}

func TestMembersInheritance(t *testing.T) {
	src := `
class Base {
	methods {
		greet() => "hi"
	}
}
class Derived {
	*Base
	name str = "?"
}
return Derived(; name="x")`
	got := memberKinds(gad.Members(eval(t, src)))
	require.Equal(t, "field", got["name"])
	require.Equal(t, "method", got["greet"], "inherited method must appear")
}
