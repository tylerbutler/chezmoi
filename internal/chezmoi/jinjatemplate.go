package chezmoi

import "fmt"

// JinjaTemplate implements TemplateEngine using gonja v2 (Jinja2 for Go).
// This is a stub that will be replaced with a full implementation.
type JinjaTemplate struct{}

// EngineType implements TemplateEngine.
func (t *JinjaTemplate) EngineType() TemplateEngineType {
	return TemplateEngineJinja
}

// Parse implements TemplateEngine.
func (t *JinjaTemplate) Parse(name string, data []byte, options TemplateOptions) error {
	return fmt.Errorf("jinja template engine not yet implemented")
}

// Execute implements TemplateEngine.
func (t *JinjaTemplate) Execute(templateData any) ([]byte, error) {
	return nil, fmt.Errorf("jinja template engine not yet implemented")
}

// AddSubTemplate implements TemplateEngine.
func (t *JinjaTemplate) AddSubTemplate(name string, engine TemplateEngine) error {
	return fmt.Errorf("jinja template engine not yet implemented")
}
