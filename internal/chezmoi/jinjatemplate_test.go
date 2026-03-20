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

func TestJinjaTemplateFilterChaining(t *testing.T) {
	engine := &JinjaTemplate{}
	err := engine.Parse("test", []byte(`{{ name | upper }}`), TemplateOptions{})
	assert.NoError(t, err)
	result, err := engine.Execute(map[string]any{"name": "alice"})
	assert.NoError(t, err)
	assert.Equal(t, "ALICE", string(result))
}

func TestJinjaTemplateDefaultFilter(t *testing.T) {
	engine := &JinjaTemplate{}
	err := engine.Parse("test", []byte(`{{ missing | default("fallback") }}`), TemplateOptions{})
	assert.NoError(t, err)
	result, err := engine.Execute(map[string]any{})
	assert.NoError(t, err)
	assert.Equal(t, "fallback", string(result))
}

func TestJinjaTemplateExpression(t *testing.T) {
	engine := &JinjaTemplate{}
	err := engine.Parse("test", []byte(`{{ "hello " ~ name }}`), TemplateOptions{})
	assert.NoError(t, err)
	result, err := engine.Execute(map[string]any{"name": "world"})
	assert.NoError(t, err)
	assert.Equal(t, "hello world", string(result))
}

func TestJinjaTemplateIncludeSubTemplate(t *testing.T) {
	// Parse the sub-template.
	sub := &JinjaTemplate{}
	err := sub.Parse("header.j2", []byte("Welcome, {{ name }}!"), TemplateOptions{})
	assert.NoError(t, err)

	// Parse the main template that includes the sub-template.
	main := &JinjaTemplate{}
	err = main.Parse("main.j2", []byte(`{% include "header.j2" %} Goodbye.`), TemplateOptions{})
	assert.NoError(t, err)

	// Register the sub-template.
	err = main.AddSubTemplate("header.j2", sub)
	assert.NoError(t, err)

	result, err := main.Execute(map[string]any{"name": "Alice"})
	assert.NoError(t, err)
	assert.Equal(t, "Welcome, Alice! Goodbye.", string(result))
}

func TestJinjaTemplateIncludeNestedPath(t *testing.T) {
	// Sub-template with a path-like name (simulating .chezmoitemplates/dir/file.j2).
	sub := &JinjaTemplate{}
	err := sub.Parse("partials/greeting.j2", []byte("Hello {{ name }}"), TemplateOptions{})
	assert.NoError(t, err)

	main := &JinjaTemplate{}
	err = main.Parse("main.j2", []byte(`{% include "partials/greeting.j2" %}`), TemplateOptions{})
	assert.NoError(t, err)

	err = main.AddSubTemplate("partials/greeting.j2", sub)
	assert.NoError(t, err)

	result, err := main.Execute(map[string]any{"name": "Bob"})
	assert.NoError(t, err)
	assert.Equal(t, "Hello Bob", string(result))
}

func TestJinjaTemplateStructFieldAccess(t *testing.T) {
	type SSHKey struct {
		Key     string
		Comment string
	}

	type SecretEntry struct {
		Notes  string
		Fields []struct {
			Name  string
			Value string
		}
	}

	funcs := template.FuncMap{
		"getKeys": func(user string) []SSHKey {
			return []SSHKey{
				{Key: "ssh-ed25519 AAAA...", Comment: user + "@work"},
				{Key: "ssh-rsa BBBB...", Comment: user + "@home"},
			}
		},
		"getSecret": func(name string) SecretEntry {
			return SecretEntry{
				Notes: "secret-note-for-" + name,
				Fields: []struct {
					Name  string
					Value string
				}{
					{Name: "public_key", Value: "ssh-ed25519 CCCC..."},
				},
			}
		},
	}

	t.Run("for_loop_with_struct_field_access", func(t *testing.T) {
		engine := &JinjaTemplate{}
		err := engine.Parse("test", []byte(
			`{% for key in getKeys("alice") %}{{ key.Key }} {{ key.Comment }}`+"\n"+`{% endfor %}`,
		), TemplateOptions{Funcs: funcs})
		assert.NoError(t, err)

		result, err := engine.Execute(map[string]any{})
		assert.NoError(t, err)
		assert.Equal(t, "ssh-ed25519 AAAA... alice@work\nssh-rsa BBBB... alice@home\n", string(result))
	})

	t.Run("struct_field_access_on_function_return", func(t *testing.T) {
		engine := &JinjaTemplate{}
		err := engine.Parse("test", []byte(
			`{{ getSecret("mykey").Notes }}`,
		), TemplateOptions{Funcs: funcs})
		assert.NoError(t, err)

		result, err := engine.Execute(map[string]any{})
		assert.NoError(t, err)
		assert.Equal(t, "secret-note-for-mykey", string(result))
	})

	t.Run("struct_slice_index_and_field_access", func(t *testing.T) {
		engine := &JinjaTemplate{}
		err := engine.Parse("test", []byte(
			`{{ getSecret("mykey").Fields[0].Value }}`,
		), TemplateOptions{Funcs: funcs})
		assert.NoError(t, err)

		result, err := engine.Execute(map[string]any{})
		assert.NoError(t, err)
		assert.Equal(t, "ssh-ed25519 CCCC...", string(result))
	})
}

func TestJinjaTemplateAddSubTemplateWrongType(t *testing.T) {
	main := &JinjaTemplate{}
	err := main.Parse("main.j2", []byte("test"), TemplateOptions{})
	assert.NoError(t, err)

	goTmpl := &GoTemplate{}
	err = goTmpl.Parse("sub", []byte("test"), TemplateOptions{})
	assert.NoError(t, err)

	err = main.AddSubTemplate("sub", goTmpl)
	assert.Error(t, err)
}
