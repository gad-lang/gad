package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWalk(t *testing.T) {
	type node struct {
		name     string
		children []*node
	}

	var (
		nd = func(name string, children ...*node) *node {
			return &node{name, children}
		}
		each = func(e *node, cb func(e *node) WalkMode) bool {
			var mode WalkMode
			for _, child := range e.children {
				mode = cb(child)
				switch mode {
				case WalkModeBreak:
					return false
				case WalkModeSkipSiblings:
					return true
				}
			}
			return true
		}
	)

	root := nd("root",
		nd("1", nd("1.1"), nd("1.2")),
		nd("2", nd("2.1"), nd("2.2", nd("2.2.1"), nd("2.2.2"), nd("2.2.3")), nd("2.3")),
		nd("3", nd("3.1"), nd("3.2"), nd("3.2.1"), nd("3.3")),
	)

	var result []string

	Walk(root, each, func(path []*node, e *node) (mode WalkMode) {
		pth := make([]string, len(path)+1)
		for i, n := range path {
			pth[i] = n.name
		}
		pth[len(path)] = e.name
		result = append(result, strings.Join(pth, "/"))
		return 0
	})

	assert.Equal(t, []string{
		"1",
		"1/1.1",
		"1/1.2",
		"2",
		"2/2.1",
		"2/2.2",
		"2/2.2/2.2.1",
		"2/2.2/2.2.2",
		"2/2.2/2.2.3",
		"2/2.3",
		"3",
		"3/3.1",
		"3/3.2",
		"3/3.2.1",
		"3/3.3",
	}, result)

	result = nil

	Walk(root, each, func(path []*node, e *node) (mode WalkMode) {
		pth := make([]string, len(path)+1)
		for i, n := range path {
			pth[i] = n.name
		}
		pth[len(path)] = e.name
		s := strings.Join(pth, "/")
		result = append(result, s)
		if e.name == "2.2" {
			return WalkModeSkipChildren
		}
		return 0
	})
	assert.Equal(t, []string{
		"1",
		"1/1.1",
		"1/1.2",
		"2",
		"2/2.1",
		"2/2.2",
		"2/2.3",
		"3",
		"3/3.1",
		"3/3.2",
		"3/3.2.1",
		"3/3.3",
	}, result)

	result = nil

	Walk(root, each, func(path []*node, e *node) (mode WalkMode) {
		pth := make([]string, len(path)+1)
		for i, n := range path {
			pth[i] = n.name
		}
		pth[len(path)] = e.name
		s := strings.Join(pth, "/")
		result = append(result, s)
		if e.name == "2.2" {
			return WalkModeSkipSiblings
		}
		return 0
	})
	assert.Equal(t, []string{
		"1",
		"1/1.1",
		"1/1.2",
		"2",
		"2/2.1",
		"2/2.2",
		"3",
		"3/3.1",
		"3/3.2",
		"3/3.2.1",
		"3/3.3",
	}, result)

	result = nil

	Walk(root, each, func(path []*node, e *node) (mode WalkMode) {
		pth := make([]string, len(path)+1)
		for i, n := range path {
			pth[i] = n.name
		}
		pth[len(path)] = e.name
		s := strings.Join(pth, "/")
		result = append(result, s)
		if e.name == "2.2" {
			return WalkModeBreak
		}
		return 0
	})
	assert.Equal(t, []string{
		"1",
		"1/1.1",
		"1/1.2",
		"2",
		"2/2.1",
		"2/2.2",
	}, result)
}
