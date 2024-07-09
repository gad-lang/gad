package utils

import (
	"fmt"
	"sort"
	"strings"
)

type Data map[any]any

func (m *Data) Set(key, value any) {
	if *m == nil {
		*m = map[any]any{}
	}
	(*m)[key] = value
}

func (m Data) GetOk(key any) (v any, ok bool) {
	v, ok = m[key]
	return
}

func (m Data) Get(key any) (v any) {
	return m[key]
}

func (m Data) Flag(key any) bool {
	return m[key] == true
}

func (m Data) String() string {
	if m == nil {
		return ""
	}
	var s []string
	for k, v := range m {
		if v == nil {
			continue
		}
		s = append(s, fmt.Sprintf("%v: %v", k, v))
	}
	sort.Strings(s)
	return strings.Join(s, ", ")
}
