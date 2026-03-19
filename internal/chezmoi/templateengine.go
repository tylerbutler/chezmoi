package chezmoi

// TemplateEngineType identifies a template engine.
type TemplateEngineType string

// Template engine types.
const (
	TemplateEngineGo    TemplateEngineType = "go"
	TemplateEngineJinja TemplateEngineType = "jinja"
)

// JinjaSuffix is the file extension for Jinja templates.
const JinjaSuffix = ".j2"

// TemplateEngine is the interface for template engines.
type TemplateEngine interface {
	// Parse parses template source bytes with the given name and options.
	Parse(name string, data []byte, options TemplateOptions) error

	// Execute executes the parsed template against the provided data context.
	// Implementations MUST deep-copy templateData before execution.
	Execute(templateData any) ([]byte, error)

	// AddSubTemplate registers a named sub-template for reuse.
	// Only templates of the same engine type can be added.
	AddSubTemplate(name string, engine TemplateEngine) error

	// EngineType returns the engine type for this instance.
	EngineType() TemplateEngineType
}
