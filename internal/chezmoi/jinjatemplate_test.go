package chezmoi

import (
	"testing"
	"text/template"

	"github.com/alecthomas/assert/v2"
)

func TestJinjaTemplateBasicExecution(t *testing.T) {
	engine := &JinjaTemplate{}
	err := engine.Parse("test", []byte("Hello {{ name }}!"), TemplateOptions{})
	assert.NoError(t, err)

	result, err := engine.Execute(map[string]any{"name": "World"})
	assert.NoError(t, err)
	assert.Equal(t, "Hello World!", string(result))
}

func TestJinjaTemplateEngineType(t *testing.T) {
	engine := &JinjaTemplate{}
	assert.Equal(t, TemplateEngineJinja, engine.EngineType())
}

func TestJinjaTemplateConditional(t *testing.T) {
	engine := &JinjaTemplate{}
	err := engine.Parse("test", []byte(`{% if os == "linux" %}yes{% else %}no{% endif %}`), TemplateOptions{})
	assert.NoError(t, err)

	result, err := engine.Execute(map[string]any{"os": "linux"})
	assert.NoError(t, err)
	assert.Equal(t, "yes", string(result))
}

func TestJinjaTemplateLoop(t *testing.T) {
	engine := &JinjaTemplate{}
	err := engine.Parse("test", []byte(`{% for item in items %}{{ item }} {% endfor %}`), TemplateOptions{})
	assert.NoError(t, err)

	result, err := engine.Execute(map[string]any{"items": []string{"a", "b", "c"}})
	assert.NoError(t, err)
	assert.Equal(t, "a b c ", string(result))
}

func TestJinjaTemplateNestedData(t *testing.T) {
	engine := &JinjaTemplate{}
	err := engine.Parse("test", []byte("{{ chezmoi.os }}"), TemplateOptions{})
	assert.NoError(t, err)

	result, err := engine.Execute(map[string]any{
		"chezmoi": map[string]any{"os": "darwin"},
	})
	assert.NoError(t, err)
	assert.Equal(t, "darwin", string(result))
}

func TestJinjaTemplateDeepCopy(t *testing.T) {
	engine := &JinjaTemplate{}
	err := engine.Parse("test", []byte("{{ name }}"), TemplateOptions{})
	assert.NoError(t, err)

	data := map[string]any{"name": "original"}
	_, err = engine.Execute(data)
	assert.NoError(t, err)
	assert.Equal(t, "original", data["name"])
}

func TestJinjaTemplateWithFunction(t *testing.T) {
	engine := &JinjaTemplate{}
	err := engine.Parse("test", []byte(`{{ greet("World") }}`), TemplateOptions{
		Funcs: template.FuncMap{
			"greet": func(name string) string { return "Hello, " + name + "!" },
		},
	})
	assert.NoError(t, err)

	result, err := engine.Execute(map[string]any{})
	assert.NoError(t, err)
	assert.Equal(t, "Hello, World!", string(result))
}

func TestJinjaTemplateDirective(t *testing.T) {
	engine := &JinjaTemplate{}
	input := "{# chezmoi:template:line-ending=crlf #}\nline1\nline2\n"
	err := engine.Parse("test", []byte(input), TemplateOptions{})
	assert.NoError(t, err)

	result, err := engine.Execute(nil)
	assert.NoError(t, err)
	assert.Equal(t, "line1\r\nline2\r\n", string(result))
}
