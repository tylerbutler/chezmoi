package chezmoi

import (
	"strings"
	"testing"
	"text/template"

	"github.com/alecthomas/assert/v2"
)

func TestWrapAsGonjaGlobals(t *testing.T) {
	funcs := template.FuncMap{
		"upper": func(s string) string { return strings.ToUpper(s) },
		"add":   func(a, b int) int { return a + b },
	}
	globals := WrapAsGonjaGlobals(funcs)
	_, ok := globals["upper"]
	assert.True(t, ok)
	_, ok = globals["add"]
	assert.True(t, ok)
}

func TestWrapAsGonjaFilters(t *testing.T) {
	funcs := template.FuncMap{
		"upper": func(s string) string { return strings.ToUpper(s) },
	}
	filters := WrapAsGonjaFilters(funcs)
	fn, ok := filters["upper"]
	assert.True(t, ok)
	assert.NotEqual(t, nil, fn)
}
