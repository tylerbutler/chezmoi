package chezmoi

import (
	"strings"
	"testing"
	"text/template"

	"github.com/alecthomas/assert/v2"
)

type contractTestCase struct {
	name          string
	goTemplate    string
	jinjaTemplate string
	data          map[string]any
	funcs         template.FuncMap
	expected      string
}

var contractTests = []contractTestCase{
	{
		name:          "simple_variable",
		goTemplate:    "{{ .name }}",
		jinjaTemplate: "{{ name }}",
		data:          map[string]any{"name": "Alice"},
		expected:      "Alice",
	},
	{
		name:          "nested_variable",
		goTemplate:    "{{ .chezmoi.os }}",
		jinjaTemplate: "{{ chezmoi.os }}",
		data:          map[string]any{"chezmoi": map[string]any{"os": "darwin"}},
		expected:      "darwin",
	},
	{
		name:          "static_text",
		goTemplate:    "hello world",
		jinjaTemplate: "hello world",
		data:          nil,
		expected:      "hello world",
	},
	{
		name:          "function_call",
		goTemplate:    `{{ upper .name }}`,
		jinjaTemplate: `{{ upper(name) }}`,
		data:          map[string]any{"name": "alice"},
		funcs:         template.FuncMap{"upper": func(s string) string { return strings.ToUpper(s) }},
		expected:      "ALICE",
	},
	{
		name:          "empty_data",
		goTemplate:    "static",
		jinjaTemplate: "static",
		data:          map[string]any{},
		expected:      "static",
	},
}

func TestContractGoTemplate(t *testing.T) {
	for _, tc := range contractTests {
		t.Run(tc.name, func(t *testing.T) {
			engine := &GoTemplate{}
			err := engine.Parse(tc.name, []byte(tc.goTemplate), TemplateOptions{Funcs: tc.funcs})
			assert.NoError(t, err)
			result, err := engine.Execute(tc.data)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, string(result))
		})
	}
}

func TestContractJinjaTemplate(t *testing.T) {
	for _, tc := range contractTests {
		t.Run(tc.name, func(t *testing.T) {
			engine := &JinjaTemplate{}
			err := engine.Parse(tc.name, []byte(tc.jinjaTemplate), TemplateOptions{Funcs: tc.funcs})
			assert.NoError(t, err)
			result, err := engine.Execute(tc.data)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, string(result))
		})
	}
}

func TestContractDeepCopy(t *testing.T) {
	for _, tc := range []struct {
		name    string
		tmpl    string
		factory func() TemplateEngine
	}{
		{"go", "{{ .name }}", func() TemplateEngine { return &GoTemplate{} }},
		{"jinja", "{{ name }}", func() TemplateEngine { return &JinjaTemplate{} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			engine := tc.factory()
			err := engine.Parse("test", []byte(tc.tmpl), TemplateOptions{})
			assert.NoError(t, err)
			data := map[string]any{"name": "original"}
			_, err = engine.Execute(data)
			assert.NoError(t, err)
			assert.Equal(t, "original", data["name"])
		})
	}
}

func TestContractPostProcessing(t *testing.T) {
	for _, tc := range []struct {
		name    string
		factory func() TemplateEngine
	}{
		{"go", func() TemplateEngine { return &GoTemplate{} }},
		{"jinja", func() TemplateEngine { return &JinjaTemplate{} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			engine := tc.factory()
			err := engine.Parse("test", []byte("line1\nline2\n"), TemplateOptions{LineEnding: "\r\n"})
			assert.NoError(t, err)
			result, err := engine.Execute(nil)
			assert.NoError(t, err)
			assert.Equal(t, "line1\r\nline2\r\n", string(result))
		})
	}
}
