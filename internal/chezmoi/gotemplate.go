package chezmoi

import (
	"encoding/json"
	"fmt"
	"maps"
	"strings"
	"text/template"

	"github.com/BurntSushi/toml"
	"github.com/goccy/go-yaml"
	"github.com/mattn/go-runewidth"
	"github.com/mitchellh/copystructure"
)

// GoTemplate implements TemplateEngine using Go's text/template.
type GoTemplate struct {
	name     string
	template *template.Template
	options  TemplateOptions
}

// EngineType implements TemplateEngine.
func (t *GoTemplate) EngineType() TemplateEngineType {
	return TemplateEngineGo
}

// Parse implements TemplateEngine.
func (t *GoTemplate) Parse(name string, data []byte, options TemplateOptions) error {
	contents, err := options.parseAndRemoveDirectives(data)
	if err != nil {
		return err
	}
	funcs := options.Funcs
	if options.FormatIndent != "" {
		funcs = maps.Clone(funcs)
		funcs["toJson"] = func(data any) string {
			var builder strings.Builder
			encoder := json.NewEncoder(&builder)
			encoder.SetIndent("", options.FormatIndent)
			if err := encoder.Encode(data); err != nil {
				panic(err)
			}
			return builder.String()
		}
		funcs["toToml"] = func(data any) string {
			var builder strings.Builder
			encoder := toml.NewEncoder(&builder)
			encoder.Indent = options.FormatIndent
			if err := encoder.Encode(data); err != nil {
				panic(err)
			}
			return builder.String()
		}
		funcs["toYaml"] = func(data any) string {
			var builder strings.Builder
			encoder := yaml.NewEncoder(&builder,
				yaml.Indent(runewidth.StringWidth(options.FormatIndent)),
			)
			if err := encoder.Encode(data); err != nil {
				panic(err)
			}
			return builder.String()
		}
	}
	tmpl, err := template.New(name).
		Option(options.Options...).
		Delims(options.LeftDelimiter, options.RightDelimiter).
		Funcs(funcs).
		Parse(string(contents))
	if err != nil {
		return err
	}
	t.name = name
	t.template = tmpl
	t.options = options
	return nil
}

// Execute implements TemplateEngine.
func (t *GoTemplate) Execute(data any) ([]byte, error) {
	if data != nil {
		// Make a deep copy of data, in case any template functions modify it.
		var err error
		data, err = copystructure.Copy(data)
		if err != nil {
			return nil, err
		}
	}

	var builder strings.Builder
	if err := t.template.ExecuteTemplate(&builder, t.name, data); err != nil {
		return nil, err
	}

	return PostProcessTemplate([]byte(builder.String()), t.options)
}

// ExecuteString executes the template and returns a string.
func (t *GoTemplate) ExecuteString(data any) (string, error) {
	resultBytes, err := t.Execute(data)
	if err != nil {
		return "", err
	}
	return string(resultBytes), nil
}

// AddSubTemplate implements TemplateEngine.
func (t *GoTemplate) AddSubTemplate(name string, engine TemplateEngine) error {
	goTmpl, ok := engine.(*GoTemplate)
	if !ok {
		return fmt.Errorf("cannot add %s template %q to Go template engine", engine.EngineType(), name)
	}
	var err error
	t.template, err = t.template.AddParseTree(goTmpl.name, goTmpl.template.Tree)
	return err
}
