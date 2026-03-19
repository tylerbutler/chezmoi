package chezmoi

import (
	"bytes"
	"crypto/sha256"
	"fmt"

	"github.com/mitchellh/copystructure"
	"github.com/nikolalohinski/gonja/v2/builtins"
	"github.com/nikolalohinski/gonja/v2/config"
	"github.com/nikolalohinski/gonja/v2/exec"
	"github.com/nikolalohinski/gonja/v2/loaders"
)

// JinjaTemplate implements TemplateEngine using gonja v2 (Jinja2 for Go).
type JinjaTemplate struct {
	name     string
	template *exec.Template
	options  TemplateOptions
	globals  map[string]any
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

	// Create a memory loader that serves our template source.
	loader, err := loaders.NewMemoryLoader(map[string]string{
		rootID: string(contents),
	})
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}

	tmpl, err := exec.NewTemplate(rootID, config.New(), loader, env)
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}

	t.name = name
	t.template = tmpl
	t.options = options
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
	_, ok := engine.(*JinjaTemplate)
	if !ok {
		return fmt.Errorf("cannot add %s template %q to Jinja template engine", engine.EngineType(), name)
	}
	// For the MVP, sub-template registration is a no-op.
	// Jinja users can use {% include %} with file system templates,
	// but dynamic sub-template registration from the .chezmoitemplates
	// directory requires a custom loader (deferred to follow-up).
	return nil
}
