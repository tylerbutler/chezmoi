package chezmoi

import (
	"bytes"
	"crypto/sha256"
	"fmt"

	"github.com/mitchellh/copystructure"
	"github.com/nikolalohinski/gonja/v2/builtins"
	"github.com/nikolalohinski/gonja/v2/config"
	"github.com/nikolalohinski/gonja/v2/exec"
)

// JinjaTemplate implements TemplateEngine using gonja v2 (Jinja2 for Go).
type JinjaTemplate struct {
	name     string
	source   string // raw template source for sub-template registration
	template *exec.Template
	options  TemplateOptions
	globals  map[string]any
	loader   *mutableLoader
}

// EngineType implements TemplateEngine.
func (t *JinjaTemplate) EngineType() TemplateEngineType {
	return TemplateEngineJinja
}

// Parse implements TemplateEngine.
func (t *JinjaTemplate) Parse(name string, data []byte, options TemplateOptions) error {
	contents, err := options.parseAndRemoveDirectives(data)
	if err != nil {
		return err
	}

	// Build globals from template functions.
	t.globals = make(map[string]any)
	if options.Funcs != nil {
		for k, v := range WrapAsGonjaGlobals(options.Funcs) {
			t.globals[k] = v
		}
	}

	// Create per-template environment to avoid global state mutation.
	filters := exec.NewFilterSet(make(map[string]exec.FilterFunction))
	filters.Update(builtins.Filters)

	// Register adapted filters from FuncMap.
	if options.Funcs != nil {
		for filterName, filterFunc := range WrapAsGonjaFilters(options.Funcs) {
			if !filters.Exists(filterName) {
				_ = filters.Register(filterName, filterFunc)
			}
		}
	}

	env := &exec.Environment{
		Context:           exec.EmptyContext().Update(builtins.GlobalFunctions).Update(builtins.GlobalVariables),
		Filters:           filters,
		Tests:             builtins.Tests,
		ControlStructures: builtins.ControlStructures,
		Methods:           builtins.Methods,
	}

	// Create a unique identifier for the template.
	rootID := fmt.Sprintf("/%s-%x", name, sha256.Sum256(contents))

	// Create a mutable loader so sub-templates can be added later via AddSubTemplate.
	loader := newMutableLoader(rootID, string(contents))

	tmpl, err := exec.NewTemplate(rootID, config.New(), loader, env)
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}

	t.name = name
	t.source = string(contents)
	t.template = tmpl
	t.options = options
	t.loader = loader
	return nil
}

// Execute implements TemplateEngine.
func (t *JinjaTemplate) Execute(data any) ([]byte, error) {
	if data != nil {
		var err error
		data, err = copystructure.Copy(data)
		if err != nil {
			return nil, err
		}
	}

	// Build context from globals + data.
	contextMap := make(map[string]any)
	for k, v := range t.globals {
		contextMap[k] = v
	}
	if m, ok := data.(map[string]any); ok {
		for k, v := range m {
			contextMap[k] = v
		}
	}

	ctx := exec.NewContext(contextMap)

	var buf bytes.Buffer
	if err := t.template.Execute(&buf, ctx); err != nil {
		return nil, fmt.Errorf("%s: %w", t.name, err)
	}

	return PostProcessTemplate(buf.Bytes(), t.options)
}

// AddSubTemplate implements TemplateEngine.
func (t *JinjaTemplate) AddSubTemplate(name string, engine TemplateEngine) error {
	jinjaTmpl, ok := engine.(*JinjaTemplate)
	if !ok {
		return fmt.Errorf("cannot add %s template %q to Jinja template engine", engine.EngineType(), name)
	}
	// Register the sub-template's source in the mutable loader so that
	// gonja's {% include "name" %} can resolve and load it at execution time.
	t.loader.Add(name, jinjaTmpl.source)
	return nil
}
