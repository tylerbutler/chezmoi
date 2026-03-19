# Jinja Template Engine Support for Chezmoi

**Date:** 2026-03-18
**Status:** Draft

## Summary

Add Jinja2 templating as a per-file alternative to Go templates in chezmoi, using the gonja library. Users opt in by using the `.j2` file extension instead of `.tmpl`. Both engines coexist and receive the same template data context and chezmoi functions.

## Motivation

Go's `text/template` syntax is unfamiliar to many users. Jinja2 is widely known from Python, Ansible, SaltStack, and other tools. Offering Jinja as an alternative lowers the barrier to entry without removing Go template support.

## Architecture

### Template Engine Interface

A new `TemplateEngine` interface abstracts over both Go templates and gonja:

```go
// internal/chezmoi/templateengine.go

type TemplateEngine interface {
    // Parse parses template source bytes with the given name and options.
    // Directives are extracted and removed before engine-specific parsing.
    Parse(name string, data []byte, options TemplateOptions) error

    // Execute executes the parsed template against the provided data context.
    // Implementations MUST deep-copy templateData before execution to prevent
    // template functions from mutating the caller's data.
    Execute(templateData any) ([]byte, error)

    // AddSubTemplate registers a named sub-template for reuse (e.g. from
    // .chezmoitemplates/). Only templates parsed by the same engine type
    // can be added. Cross-engine includes are not supported.
    AddSubTemplate(name string, engine TemplateEngine) error

    // EngineType returns the engine type for this instance.
    EngineType() TemplateEngineType
}
```

**Key contracts:**
- `Execute` accepts `any` (not `map[string]any`) to match existing callers. Implementations must deep-copy the data before execution to prevent mutation.
- `AddSubTemplate` accepts another `TemplateEngine` instance. Implementations should type-assert to their concrete type and return an error if the engine type doesn't match.
- `EngineType` enables runtime engine-type checking for sub-template filtering and diagnostics.

### Engine Type Selection

A new `TemplateEngineType` determines which engine processes a file:

```go
type TemplateEngineType string

const (
    TemplateEngineGo    TemplateEngineType = "go"
    TemplateEngineJinja TemplateEngineType = "jinja"
)

// JinjaSuffix is the file extension for Jinja templates.
const JinjaSuffix = ".j2"
```

Selection is based on file extension:
- `.tmpl` (existing `TemplateSuffix`) → `TemplateEngineGo`
- `.j2` (`JinjaSuffix`) → `TemplateEngineJinja`

### FileAttr Changes

`FileAttr` retains `Template bool` and gains `TemplateEngine TemplateEngineType`:

```go
type FileAttr struct {
    // ... existing fields ...
    Template       bool
    TemplateEngine TemplateEngineType  // "go" (default when Template=true) or "jinja"
}
```

**`parseFileAttr` changes** (`attr.go`): The suffix-stripping switch adds a `.j2` case:

```go
case strings.HasSuffix(name, JinjaSuffix):
    name = strings.TrimSuffix(name, JinjaSuffix)
    fileAttr.Template = true
    fileAttr.TemplateEngine = TemplateEngineJinja
case strings.HasSuffix(name, TemplateSuffix):
    name = strings.TrimSuffix(name, TemplateSuffix)
    fileAttr.Template = true
    fileAttr.TemplateEngine = TemplateEngineGo
```

When `Template` is true and `TemplateEngine` is empty (zero value), it defaults to `TemplateEngineGo` for backward compatibility.

**`SourceName()` changes** (`attr.go`): Uses `TemplateEngine` field to choose suffix:

```go
func (fa FileAttr) SourceName() string {
    // ... existing logic ...
    if fa.Template {
        switch fa.TemplateEngine {
        case TemplateEngineJinja:
            sourceName += JinjaSuffix
        default:
            sourceName += TemplateSuffix
        }
    }
    return sourceName
}
```

**Duplicate target detection:** Having both `dot_bashrc.tmpl` and `dot_bashrc.j2` in the same source directory is a duplicate target error, consistent with existing chezmoi behavior for conflicting source entries targeting the same destination path.

### ExecuteTemplateData Changes

`ExecuteTemplateDataOptions` gains an engine type field:

```go
type ExecuteTemplateDataOptions struct {
    NameRelPath RelPath
    Data        []byte
    DestAbsPath AbsPath
    ExtraData   map[string]any
    EngineType  TemplateEngineType  // new field
}
```

All call sites in `sourcestate.go` that check `fileAttr.Template` pass `fileAttr.TemplateEngine` via `options.EngineType`. `ExecuteTemplateData()` uses the factory to create the appropriate engine:

```go
func (s *SourceState) ExecuteTemplateData(options ExecuteTemplateDataOptions) ([]byte, error) {
    engineType := options.EngineType
    if engineType == "" {
        engineType = TemplateEngineGo  // backward-compatible default
    }
    engine := NewTemplateEngine(engineType)
    // ... parse, add sub-templates (filtered by engine type), execute ...
}
```

### Factory

```go
func NewTemplateEngine(engineType TemplateEngineType) TemplateEngine {
    switch engineType {
    case TemplateEngineJinja:
        return &JinjaTemplate{}
    default:
        return &GoTemplate{}
    }
}
```

### GoTemplate (Refactored)

The existing `Template` struct is renamed to `GoTemplate` and implements `TemplateEngine`. Logic is split:

- **Engine-specific:** Parsing and execution stay in `GoTemplate`.
- **Shared post-processing:** Line ending normalization and encoding conversion are extracted to a `PostProcessTemplate()` helper used by both engines.

The refactoring is a pure reshaping — identical behavior, verified by existing tests passing before and after. The `Template` type is internal-only (in `internal/chezmoi`), so no public API is affected. Test files referencing `Template` will be updated to `GoTemplate`.

### Shared Templates (`.chezmoitemplates/`)

Templates in `.chezmoitemplates/` are parsed by the engine matching their file extension:
- `.chezmoitemplates/header.tmpl` → parsed by Go template engine
- `.chezmoitemplates/header.j2` → parsed by Jinja engine
- `.chezmoitemplates/header` (no extension) → parsed by Go template engine (backward compatible default)

**`addTemplatesDir` changes** (`sourcestate.go`): When iterating files in `.chezmoitemplates/`, inspect the extension to determine engine type. Use the factory to create the appropriate engine, parse the file, and store in the templates map.

**Map key strategy:** Template names in `s.templates` **retain their extension** (e.g. `"header.tmpl"`, `"header.j2"`). This means:
- Go templates: `{{ template "header.tmpl" . }}`
- Jinja templates: `{% include "header.j2" %}`
- Templates without an extension keep their name as-is (backward compatible)

This avoids collisions when `header.tmpl` and `header.j2` coexist, and makes the engine type explicit in the include statement.

`SourceState.templates` changes from `map[string]*Template` to `map[string]TemplateEngine`. When `ExecuteTemplateData()` adds sub-templates, it only adds templates whose `EngineType()` matches the current file's engine.

**Cross-engine includes are not supported.** If a `.j2` file tries to include a Go-only template, execution returns an error explaining the limitation.

### Special Files

The following special files currently support `.tmpl` suffix and must also support `.j2`:

- `.chezmoiignore` / `.chezmoiignore.tmpl` → also `.chezmoiignore.j2`
- `.chezmoiremove` / `.chezmoiremove.tmpl` → also `.chezmoiremove.j2`
- `.chezmoiversion`

**Changes required:**
- `specialDirEntryNames` in `chezmoi.go` (lines 86-103): Add `.j2` variants for each special file that currently has a `.tmpl` variant.
- Name-matching in `sourcestate.go` (e.g. line 1076-1078): Add `.j2` suffix checks alongside existing `.tmpl` checks.

### JinjaTemplate (New)

```go
// internal/chezmoi/jinjatemplate.go

type JinjaTemplate struct {
    name     string
    env      *gonja.Environment
    template *gonja.Template
    options  TemplateOptions
}

func (t *JinjaTemplate) EngineType() TemplateEngineType {
    return TemplateEngineJinja
}
```

Key behaviors:

- **Directive parsing:** Uses the same `templateDirectiveRx` regex as Go templates. The regex is line-based and matches `chezmoi:template:` anywhere on a line, so `{# chezmoi:template: encoding="utf-8" #}` is matched and the entire line is removed. No engine-specific regex needed.
- **Go-specific directives:** `left-delimiter`, `right-delimiter`, and `missing-key` are Go-template-specific. If these appear in a `.j2` file's directive, a warning is logged and they are ignored. All other directives (`encoding`, `format-indent`, `line-ending`) apply to both engines.
- **Template data:** Passed as gonja context. `.chezmoi.os` becomes `{{ chezmoi.os }}` (no leading dot — natural Jinja syntax).
- **Named templates:** Templates from `.chezmoitemplates/` (with `.j2` extension) are loaded into the gonja environment. Supports `{% include "name.j2" %}` and `{% extends "base.j2" %}`.
- **Error handling:** Gonja errors wrapped to match chezmoi's reporting format (file, line, description).
- **Post-processing:** Line endings and encoding applied via the shared `PostProcessTemplate()` helper.

### Function Adapter Layer

Chezmoi's 80+ template functions are defined as `template.FuncMap` entries. The adapter converts these for gonja:

```go
// internal/chezmoi/funcadapter.go

func WrapAsGonjaGlobals(funcs template.FuncMap) map[string]*exec.Value
func WrapAsGonjaFilters(funcs template.FuncMap) map[string]FilterFunc
```

**`FormatIndent` handling:** When `FormatIndent` is set in `TemplateOptions`, the Go template engine currently clones the FuncMap and replaces `toJson`/`toToml`/`toYaml` with indented versions. This override is applied to the FuncMap *before* it is passed to `WrapAsGonjaGlobals`/`WrapAsGonjaFilters`, so the adapter always receives the final function implementations. Both engines share this pre-processing step.

**Filter classification rule:** A function becomes a Jinja filter if and only if its first parameter is the "primary input" (the value being transformed). Specifically:
- **Filter + global:** Functions with signature `func(input, ...args) result` where the first arg is the data being transformed (e.g. `toJson`, `comment`, `quote`, `hexEncode`, `replaceAllRegex`).
- **Global only:** Functions that fetch/produce data rather than transform it (e.g. `output`, `onepassword`, `include`, `glob`, `joinPath`), and functions with no parameters or only configuration parameters.

| Function type | Go template | Jinja global | Jinja filter |
|---|---|---|---|
| Transforms (`toJson`) | `{{ toJson .data }}` | `{{ toJson(data) }}` | `{{ data \| toJson }}` |
| Fetchers (`output`) | `{{ output "cmd" }}` | `{{ output("cmd") }}` | n/a |
| Secret managers (`onepassword`) | `{{ onepassword "item" }}` | `{{ onepassword("item") }}` | n/a |

**MVP scope (Tier 1 + 2):**
- Data access (template data context — native)
- String/data transforms: `toJson`, `fromYaml`, `toToml`, `fromJson`, `comment`, `quote`, etc.
- File operations: `include`, `includeTemplate`, `glob`, `stat`
- Execution: `output`, `outputList`
- Utility: `joinPath`, `lookPath`, `findExecutable`

**Deferred (Tier 3):** Secret manager integrations (~30 functions). The adapter supports them; adding them is incremental work.

## User Experience

### What Jinja users gain

- `{% if os == "linux" %}` instead of `{{ if eq .os "linux" }}`
- `{% for k, v in data.items() %}` instead of `{{ range $k, $v := .data }}`
- `{{ value | default("fallback") }}` filter chaining
- `{% extends "base.j2" %}` / `{% block name %}` template inheritance
- Expressions: `{{ "hello " + name }}` instead of `{{ printf "hello %s" .name }}`

### File naming

```
# Go template (unchanged)
dot_bashrc.tmpl

# Jinja template (new)
dot_bashrc.j2
```

Both are valid. A user can mix `.tmpl` and `.j2` files in the same source directory, as long as they don't target the same destination (which would be a duplicate target error).

### Scope of `.j2` support

The `.j2` extension is supported everywhere `.tmpl` is supported:
- Managed dotfiles (e.g. `dot_bashrc.j2`)
- `.chezmoitemplates/` shared templates
- `.chezmoiignore.j2`
- `.chezmoiremove.j2`
- Any other file that currently supports `.tmpl` suffix

### CLI behavior

- `chezmoi add --template` continues to produce `.tmpl` files (backward compatible default).
- A new `--template-engine=jinja` flag (or `--jinja` shorthand) causes `chezmoi add` to produce `.j2` files instead.
- A config option `templateEngine: jinja` in `.chezmoi.toml` sets the default engine for `chezmoi add --template` only. It does **not** change how existing files are processed — files are always processed based on their actual extension.
- All existing commands (`edit`, `diff`, `apply`, `cat`, etc.) work with `.j2` files without changes.

### Diagnostics

`chezmoi doctor` gains a Jinja engine health check:
- Reports gonja version/availability.
- If the user has `.j2` files in their source directory, the check is mandatory. Otherwise it's informational.

## Testing Strategy

### TDD order

1. **Interface contract tests** — Shared test suite both engines must pass. Covers template data access, function calls, error handling, post-processing, deep-copy behavior.
2. **Go template refactor** — Existing tests + contract tests green after `Template` → `GoTemplate`.
3. **Jinja-specific tests** (red) — Filter chaining, template inheritance, expressions, blocks.
4. **JinjaTemplate implementation** (green).
5. **Adapter tests** (red) — Each wrapped function produces correct output via gonja calling convention, both as global and filter.
6. **Adapter implementation** (green).
7. **FileAttr tests** (red) — `.j2` extension parsed, stripped, mapped to `TemplateEngineJinja`. `SourceName()` round-trips correctly.
8. **FileAttr implementation** (green).
9. **Special file tests** (red) — `.chezmoiignore.j2` and `.chezmoiremove.j2` recognized.
10. **Special file implementation** (green).
11. **Integration tests** — End-to-end: `.j2` source file → correct target file.

### Test layers

- **Contract tests:** Behavioral parity between engines on shared functionality.
- **Regression tests:** All existing Go template tests pass unchanged.
- **Engine-specific tests:** Jinja features not present in Go templates.
- **Adapter tests:** Function wrapping correctness for both invocation styles.
- **FileAttr tests:** Extension parsing, suffix stripping, `SourceName()` round-trip.
- **Integration tests:** Full pipeline from source file to target file.

## Dependencies

### gonja

- **Repository:** [github.com/noirbizarre/gonja](https://github.com/noirbizarre/gonja)
- **Description:** Go implementation of Jinja2 templates, forked from pongo2 lineage with improved Jinja2 fidelity.
- **Known limitations to evaluate:** Macro support depth, custom extension loading, error message quality. These should be assessed during TDD step 3 (Jinja-specific tests) — if a limitation is blocking, it will surface as a failing test before any production code is written.
- **Mitigation:** If gonja proves insufficient, pongo2 is a fallback. The `TemplateEngine` interface makes swapping libraries a contained change.

## Files to Create/Modify

### New files
- `internal/chezmoi/templateengine.go` — Interface, type constants, factory
- `internal/chezmoi/gotemplate.go` — Refactored Go template engine
- `internal/chezmoi/jinjatemplate.go` — Jinja engine implementation
- `internal/chezmoi/funcadapter.go` — Function adapter layer
- `internal/chezmoi/postprocess.go` — Shared post-processing helpers
- `internal/chezmoi/templateengine_test.go` — Contract tests
- `internal/chezmoi/gotemplate_test.go` — Go engine tests (migrated from template_test.go)
- `internal/chezmoi/jinjatemplate_test.go` — Jinja engine tests
- `internal/chezmoi/funcadapter_test.go` — Adapter tests

### Modified files
- `internal/chezmoi/template.go` — Removed (logic moves to `gotemplate.go`)
- `internal/chezmoi/sourcestate.go` — `ExecuteTemplateData()` uses factory + interface; `ExecuteTemplateDataOptions` gains `EngineType`; `s.templates` becomes `map[string]TemplateEngine`; `addTemplatesDir` determines engine from extension; `FileAttr` parsing recognizes `.j2`
- `internal/chezmoi/attr.go` — `FileAttr` gains `TemplateEngine` field; `parseFileAttr` handles `.j2`; `SourceName()` uses engine type for suffix
- `internal/chezmoi/chezmoi.go` — `specialDirEntryNames` gains `.j2` variants
- `internal/cmd/config.go` — `FormatIndent` FuncMap override extracted to shared helper
- `go.mod` / `go.sum` — Add gonja dependency

## Risks

- **gonja compatibility:** gonja may not support all Jinja2 features. Mitigated by testing early (TDD step 3) and documenting limitations. The interface makes library swaps contained.
- **Go template regression:** Refactoring the existing template path risks breaking things. Mitigated by running all existing tests before and after, plus contract tests.
- **Function adapter edge cases:** Some chezmoi functions have complex signatures (variadic args, optional params). Adapter must handle these correctly. Mitigated by per-function adapter tests.
- **Performance:** gonja may be slower than Go's `text/template`. Acceptable for a dotfile manager — templates are small and infrequently executed.
- **Cross-engine confusion:** Users may not understand why `{% include "header" %}` fails when `header` is a `.tmpl` template. Mitigated by clear error messages explaining engine type mismatch.
- **Duplicate target conflicts:** Users might accidentally create both `dot_bashrc.tmpl` and `dot_bashrc.j2`. Mitigated by chezmoi's existing duplicate target detection producing a clear error.
