# Jinja Template Engine Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add Jinja2 templating as a per-file alternative to Go templates in chezmoi, using the gonja v2 library.

**Architecture:** Introduce a `TemplateEngine` interface that both `GoTemplate` and `JinjaTemplate` implement. The existing `Template` struct is refactored into `GoTemplate`. File extension (`.tmpl` vs `.j2`) selects the engine. A function adapter layer wraps chezmoi's `template.FuncMap` entries for gonja.

**Tech Stack:** Go 1.25, gonja v2 (`github.com/nikolalohinski/gonja/v2`), text/template (existing)

**Spec:** `docs/superpowers/specs/2026-03-18-jinja-template-engine-design.md`

**Deferred to follow-up:** CLI changes (`--template-engine=jinja` flag, config option), `chezmoi doctor` health check, Tier 3 secret manager functions.

---

### Task 1: Add gonja v2 dependency

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`

- [ ] **Step 1: Add gonja v2 module**

```bash
cd /Volumes/Code/claude-workspace-ccl/chezmoi && go get github.com/nikolalohinski/gonja/v2@latest
```

- [ ] **Step 2: Verify it resolves**

Run: `cd /Volumes/Code/claude-workspace-ccl/chezmoi && go mod tidy`
Expected: clean exit, no errors

- [ ] **Step 3: Commit**

```bash
cd /Volumes/Code/claude-workspace-ccl/chezmoi
git add go.mod go.sum
git commit -m "$(cat <<'EOF'
chore: add gonja v2 dependency

Add github.com/nikolalohinski/gonja/v2 for Jinja2 template support.
EOF
)"
```

---

### Task 2: Define TemplateEngine interface and types

**Files:**
- Create: `internal/chezmoi/templateengine.go`
- Create: `internal/chezmoi/templateengine_test.go`

- [ ] **Step 1: Write tests for type constants**

Create `internal/chezmoi/templateengine_test.go`:

```go
package chezmoi

import (
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestTemplateEngineType(t *testing.T) {
	assert.Equal(t, TemplateEngineType("go"), TemplateEngineGo)
	assert.Equal(t, TemplateEngineType("jinja"), TemplateEngineJinja)
}

func TestJinjaSuffix(t *testing.T) {
	assert.Equal(t, ".j2", JinjaSuffix)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Volumes/Code/claude-workspace-ccl/chezmoi && go test ./internal/chezmoi/ -run TestTemplateEngineType -v`
Expected: FAIL — types not defined

- [ ] **Step 3: Write the interface and types**

Create `internal/chezmoi/templateengine.go`:

```go
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

// NewTemplateEngine creates a TemplateEngine of the given type.
func NewTemplateEngine(engineType TemplateEngineType) TemplateEngine {
	switch engineType {
	case TemplateEngineJinja:
		return &JinjaTemplate{}
	default:
		return &GoTemplate{}
	}
}
```

Note: This will not compile yet until `GoTemplate` and `JinjaTemplate` exist. That's fine — the types test will pass once the interface file compiles with the constants.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Volumes/Code/claude-workspace-ccl/chezmoi && go test ./internal/chezmoi/ -run "TestTemplateEngineType|TestJinjaSuffix" -v`
Expected: May fail if `GoTemplate`/`JinjaTemplate` don't exist yet. If so, temporarily comment out `NewTemplateEngine` and add it back in Task 4.

- [ ] **Step 5: Commit**

```bash
cd /Volumes/Code/claude-workspace-ccl/chezmoi
git add internal/chezmoi/templateengine.go internal/chezmoi/templateengine_test.go
git commit -m "$(cat <<'EOF'
feat: add TemplateEngine interface and types

Define the TemplateEngine interface, TemplateEngineType constants,
JinjaSuffix, and NewTemplateEngine factory.
EOF
)"
```

---

### Task 3: Extract shared post-processing helpers

**Files:**
- Create: `internal/chezmoi/postprocess.go`
- Create: `internal/chezmoi/postprocess_test.go`

- [ ] **Step 1: Write tests for PostProcessTemplate**

Create `internal/chezmoi/postprocess_test.go`:

```go
package chezmoi

import (
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestPostProcessTemplate(t *testing.T) {
	for _, tc := range []struct {
		name     string
		input    []byte
		options  TemplateOptions
		expected string
	}{
		{
			name:     "no_processing",
			input:    []byte("hello"),
			options:  TemplateOptions{},
			expected: "hello",
		},
		{
			name:  "line_ending_crlf",
			input: []byte("line1\nline2\n"),
			options: TemplateOptions{
				LineEnding: "\r\n",
			},
			expected: "line1\r\nline2\r\n",
		},
		{
			name:  "line_ending_lf",
			input: []byte("line1\r\nline2\r\n"),
			options: TemplateOptions{
				LineEnding: "\n",
			},
			expected: "line1\nline2\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := PostProcessTemplate(tc.input, tc.options)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, string(actual))
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Volumes/Code/claude-workspace-ccl/chezmoi && go test ./internal/chezmoi/ -run TestPostProcessTemplate -v`
Expected: FAIL — `PostProcessTemplate` not defined

- [ ] **Step 3: Write PostProcessTemplate**

Create `internal/chezmoi/postprocess.go`:

```go
package chezmoi

// PostProcessTemplate applies line ending normalization and encoding
// conversion to template output.
func PostProcessTemplate(data []byte, options TemplateOptions) ([]byte, error) {
	result := []byte(replaceLineEndings(string(data), options.LineEnding))
	if options.Encoding != nil {
		return options.Encoding.NewEncoder().Bytes(result)
	}
	return result, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Volumes/Code/claude-workspace-ccl/chezmoi && go test ./internal/chezmoi/ -run TestPostProcessTemplate -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd /Volumes/Code/claude-workspace-ccl/chezmoi
git add internal/chezmoi/postprocess.go internal/chezmoi/postprocess_test.go
git commit -m "$(cat <<'EOF'
refactor: extract PostProcessTemplate helper

Extract line ending normalization and encoding conversion into a shared
helper for use by both Go and Jinja template engines.
EOF
)"
```

---

### Task 4: Refactor Template → GoTemplate

This is a pure refactor. All existing tests must continue to pass.

**Files:**
- Create: `internal/chezmoi/gotemplate.go`
- Modify: `internal/chezmoi/template.go` — remove struct/methods, keep TemplateOptions and helpers
- Modify: `internal/chezmoi/sourcestate.go` — update `templates` map type, `ExecuteTemplateData`, `addTemplatesDir`
- Modify: `internal/chezmoi/externaldiffsystem.go` — `*Template` → `*GoTemplate`
- Modify: `internal/cmd/config.go` — `*Template` → `*GoTemplate`
- Modify: `internal/cmd/mergecmd.go` — `*Template` → `*GoTemplate`
- Modify: `internal/chezmoi/sourcestate_test.go` — `map[string]*Template` → `map[string]TemplateEngine`

**Key callers of `*Template` / `ParseTemplate` / `ExecuteString` that need updating:**
- `internal/chezmoi/sourcestate.go:151` — `templates map[string]*Template`
- `internal/chezmoi/sourcestate.go:870-900` — `ExecuteTemplateData` uses `ParseTemplate`, `AddParseTree`
- `internal/chezmoi/sourcestate.go:1628-1636` — `addTemplatesDir` stores `*Template`
- `internal/chezmoi/externaldiffsystem.go:217-222` — calls `ParseTemplate`, `ExecuteString`
- `internal/cmd/config.go:955` — calls `ParseTemplate`
- `internal/cmd/config.go:1798` — calls `ParseTemplate`
- `internal/cmd/mergecmd.go:150-157` — calls `ParseTemplate`, `ExecuteString`
- `internal/cmd/templatefuncs.go:266` — `includeTemplate` calls `ParseTemplate` and `.Execute`
- `internal/chezmoi/sourcestate_test.go:1486,2235` — `map[string]*Template`

**Note on `includeTemplate`:** The `includeTemplate` function in `templatefuncs.go` hardcodes Go template parsing. For now this is acceptable — `includeTemplate` in `.j2` files will parse the included file as Go templates, which is consistent with the MVP scope (Jinja users use `{% include %}` instead). A future task should make `includeTemplate` engine-aware.

- [ ] **Step 1: Create gotemplate.go with GoTemplate implementing TemplateEngine**

Create `internal/chezmoi/gotemplate.go`:

```go
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
// This is a convenience method used by externaldiffsystem and mergecmd.
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
```

- [ ] **Step 2: Update template.go — keep only TemplateOptions and helpers, add backward-compat wrapper**

Remove from `internal/chezmoi/template.go`:
- `Template` struct (lines 21-25)
- `ParseTemplate` function (lines 40-90) — replace with wrapper
- `AddParseTree` method (lines 93-97)
- `Execute` method (lines 100-120)
- `ExecuteString` method (lines 123-129)
- Imports only needed by removed code: `json`, `maps`, `strings`, `template`, `toml`, `yaml`, `runewidth`, `copystructure`

Keep:
- `TemplateOptions` struct (lines 28-36)
- `parseAndRemoveDirectives` method (lines 134-194)
- `removeMatches` function (lines 197-205)
- `replaceLineEndings` function (lines 209-214)

Add backward-compatible wrapper:

```go
// ParseTemplate parses a template and returns a GoTemplate.
func ParseTemplate(name string, data []byte, options TemplateOptions) (*GoTemplate, error) {
	t := &GoTemplate{}
	if err := t.Parse(name, data, options); err != nil {
		return nil, err
	}
	return t, nil
}
```

- [ ] **Step 3: Update sourcestate.go**

**Line 151:** `templates map[string]*Template` → `templates map[string]TemplateEngine`

**Lines 870-900 (`ExecuteTemplateData`):** Update to use interface:

```go
func (s *SourceState) ExecuteTemplateData(options ExecuteTemplateDataOptions) ([]byte, error) {
	templateOptions := options.TemplateOptions
	templateOptions.Funcs = s.templateFuncs
	templateOptions.Options = slices.Clone(s.templateOptions)

	tmpl, err := ParseTemplate(options.NameRelPath.String(), options.Data, templateOptions)
	if err != nil {
		return nil, err
	}

	for name, t := range s.templates {
		if err := tmpl.AddSubTemplate(name, t); err != nil {
			return nil, err
		}
	}

	templateData := s.TemplateData()
	if chezmoiTemplateData, ok := templateData["chezmoi"].(map[string]any); ok {
		chezmoiTemplateData["sourceFile"] = options.NameRelPath.String()
		chezmoiTemplateData["targetFile"] = options.DestAbsPath.String()
	}
	RecursiveMerge(templateData, options.ExtraData)

	result, err := tmpl.Execute(templateData)
	if errors.Is(err, errReturnEmpty) {
		return nil, nil
	}
	return result, err
}
```

**Find and update all `map[string]*Template` and `make(map[string]*Template)` in sourcestate.go** → `map[string]TemplateEngine` / `make(map[string]TemplateEngine)`

- [ ] **Step 4: Update external callers**

**`internal/chezmoi/externaldiffsystem.go`:** Change `*Template` to `*GoTemplate` at line 217 and wherever the type is referenced. `ExecuteString` is preserved on `GoTemplate`.

**`internal/cmd/config.go`:** Change `*Template` to `*GoTemplate` at lines 955, 1798. The `ParseTemplate` wrapper returns `*GoTemplate` so this may just work.

**`internal/cmd/mergecmd.go`:** Change `*Template` to `*GoTemplate` at line 150. `ExecuteString` is preserved on `GoTemplate`.

- [ ] **Step 5: Update sourcestate_test.go**

Change `map[string]*Template` to `map[string]TemplateEngine` at lines 1486, 2235 and in the `withTemplates` helper.

- [ ] **Step 6: Run all tests**

Run: `cd /Volumes/Code/claude-workspace-ccl/chezmoi && go build ./...`
Expected: Clean compile

Run: `cd /Volumes/Code/claude-workspace-ccl/chezmoi && go test ./internal/chezmoi/ -v -count=1 2>&1 | tail -20`
Expected: All PASS

Run: `cd /Volumes/Code/claude-workspace-ccl/chezmoi && go test ./... -count=1 2>&1 | tail -30`
Expected: All PASS

- [ ] **Step 7: Commit**

```bash
cd /Volumes/Code/claude-workspace-ccl/chezmoi
git add internal/chezmoi/gotemplate.go internal/chezmoi/template.go internal/chezmoi/sourcestate.go internal/chezmoi/sourcestate_test.go internal/chezmoi/externaldiffsystem.go internal/cmd/config.go internal/cmd/mergecmd.go
git commit -m "$(cat <<'EOF'
refactor: rename Template to GoTemplate behind TemplateEngine interface

Refactor the existing Template struct into GoTemplate implementing the
new TemplateEngine interface. All external callers updated. The templates
map in SourceState now stores TemplateEngine values.
EOF
)"
```

---

### Task 5: Add TemplateEngine field to FileAttr

**Files:**
- Modify: `internal/chezmoi/attr.go`
- Modify: `internal/chezmoi/attr_test.go`
- Modify: `internal/chezmoi/chezmoi.go`

- [ ] **Step 1: Write failing test for .j2 parsing**

Add test cases to `TestFileAttrLiteral` in `internal/chezmoi/attr_test.go`:

```go
{
    sourceName: "file.j2",
    fileAttr: FileAttr{
        TargetName:     "file",
        Type:           SourceFileTypeFile,
        Template:       true,
        TemplateEngine: TemplateEngineJinja,
    },
},
{
    sourceName: "dot_bashrc.j2",
    fileAttr: FileAttr{
        TargetName:     ".bashrc",
        Type:           SourceFileTypeFile,
        Template:       true,
        TemplateEngine: TemplateEngineJinja,
    },
},
{
    sourceName: "executable_dot_script.j2",
    fileAttr: FileAttr{
        TargetName:     ".script",
        Type:           SourceFileTypeFile,
        Executable:     true,
        Template:       true,
        TemplateEngine: TemplateEngineJinja,
    },
},
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Volumes/Code/claude-workspace-ccl/chezmoi && go test ./internal/chezmoi/ -run TestFileAttrLiteral -v`
Expected: FAIL

- [ ] **Step 3: Add TemplateEngine field to FileAttr and update parsing**

In `internal/chezmoi/attr.go`:

Add field to `FileAttr` struct:
```go
type FileAttr struct {
	TargetName     string
	Type           SourceFileTargetType
	Condition      ScriptCondition
	Empty          bool
	Encrypted      bool
	Executable     bool
	Order          ScriptOrder
	Private        bool
	ReadOnly       bool
	Template       bool
	TemplateEngine TemplateEngineType
}
```

In `parseFileAttr`, add `templateEngine` to local variables:
```go
var templateEngine TemplateEngineType
```

Update the suffix switch to recognize `.j2` (before the `.tmpl` case so `.j2` doesn't accidentally match `.tmpl` logic):
```go
	switch {
	case strings.HasSuffix(name, literalSuffix):
		name = name[:len(name)-len(literalSuffix)]
	case strings.HasSuffix(name, JinjaSuffix):
		name = name[:len(name)-len(JinjaSuffix)]
		template = true
		templateEngine = TemplateEngineJinja
		name, _ = strings.CutSuffix(name, literalSuffix)
	case strings.HasSuffix(name, TemplateSuffix):
		name = name[:len(name)-len(TemplateSuffix)]
		template = true
		templateEngine = TemplateEngineGo
		name, _ = strings.CutSuffix(name, literalSuffix)
	}
```

Add `TemplateEngine` to return value:
```go
	return FileAttr{
		// ... existing fields ...
		Template:       template,
		TemplateEngine: templateEngine,
	}, nil
```

Update `SourceName` to use engine type for suffix:
```go
	if fa.Template {
		switch fa.TemplateEngine {
		case TemplateEngineJinja:
			sourceName += JinjaSuffix
		default:
			sourceName += TemplateSuffix
		}
	}
```

Update `LogValue` to include TemplateEngine:
```go
slog.String("TemplateEngine", string(fa.TemplateEngine)),
```

- [ ] **Step 4: Update fileSuffixRx in chezmoi.go**

In `internal/chezmoi/chezmoi.go` line 80:
```go
fileSuffixRx = regexp.MustCompile(`\.(j2|literal|tmpl)\z`)
```

- [ ] **Step 5: Fix combinatorial TestFileAttr**

The combinatorial `TestFileAttr` generates all permutations including `Template: true`. After Task 4, `parseFileAttr` now sets `TemplateEngine: TemplateEngineGo` when `.tmpl` is detected. The generated `FileAttr` must match. Add a post-generation fixup after each `combinator.Generate` block that includes `Template`:

```go
for i := range fileAttrs {
    if fileAttrs[i].Template {
        fileAttrs[i].TemplateEngine = TemplateEngineGo
    }
}
```

- [ ] **Step 6: Run all tests**

Run: `cd /Volumes/Code/claude-workspace-ccl/chezmoi && go test ./internal/chezmoi/ -run "TestFileAttr|TestInvalidFileAttr" -v`
Expected: All PASS

Run: `cd /Volumes/Code/claude-workspace-ccl/chezmoi && go test ./internal/chezmoi/ -v -count=1 2>&1 | tail -20`
Expected: All PASS

- [ ] **Step 7: Commit**

```bash
cd /Volumes/Code/claude-workspace-ccl/chezmoi
git add internal/chezmoi/attr.go internal/chezmoi/attr_test.go internal/chezmoi/chezmoi.go
git commit -m "$(cat <<'EOF'
feat: add .j2 file extension support to FileAttr

FileAttr gains a TemplateEngine field. parseFileAttr recognizes .j2
as Jinja templates. SourceName round-trips correctly for both engines.
EOF
)"
```

---

### Task 6: Update special files and knownPrefixedFiles for .j2

**Files:**
- Modify: `internal/chezmoi/chezmoi.go` — add .j2 variants to `knownPrefixedFiles`
- Modify: `internal/chezmoi/sourcestate.go` — recognize `.chezmoiignore.j2`, `.chezmoiremove.j2`
- Modify: `internal/chezmoi/format.go` — add `isPrefixDotFormatDotJ2`

- [ ] **Step 1: Update knownPrefixedFiles**

In `internal/chezmoi/chezmoi.go`, add `.j2` variants alongside each `.tmpl` variant:

```go
var knownPrefixedFiles = chezmoiset.New(
	Prefix+".json"+TemplateSuffix,
	Prefix+".json"+JinjaSuffix,
	Prefix+".toml"+TemplateSuffix,
	Prefix+".toml"+JinjaSuffix,
	Prefix+".yaml"+TemplateSuffix,
	Prefix+".yaml"+JinjaSuffix,
	RootName,
	VersionName,
	dataName+".json",
	dataName+".toml",
	dataName+".yaml",
	externalName+".json"+TemplateSuffix,
	externalName+".json"+JinjaSuffix,
	externalName+".json",
	externalName+".toml"+TemplateSuffix,
	externalName+".toml"+JinjaSuffix,
	externalName+".toml",
	externalName+".yaml"+TemplateSuffix,
	externalName+".yaml"+JinjaSuffix,
	externalName+".yaml",
	ignoreName+TemplateSuffix,
	ignoreName+JinjaSuffix,
	ignoreName,
	removeName+TemplateSuffix,
	removeName+JinjaSuffix,
	removeName,
)
```

- [ ] **Step 2: Update sourcestate.go ignore/remove recognition**

In `internal/chezmoi/sourcestate.go` (around line 1076):

```go
case fileInfo.Name() == ignoreName || fileInfo.Name() == ignoreName+TemplateSuffix || fileInfo.Name() == ignoreName+JinjaSuffix:
	return s.addPatterns(s.ignore, sourceAbsPath, parentSourceRelPath)
case fileInfo.Name() == removeName || fileInfo.Name() == removeName+TemplateSuffix || fileInfo.Name() == removeName+JinjaSuffix:
	return s.addPatterns(s.remove, sourceAbsPath, parentSourceRelPath)
```

- [ ] **Step 3: Add isPrefixDotFormatDotJ2 and update external file recognition**

In `internal/chezmoi/format.go`, add after `isPrefixDotFormatDotTmpl`:

```go
func isPrefixDotFormatDotJ2(name, prefix string) bool {
	for extension := range FormatsByExtension {
		if name == prefix+"."+extension+JinjaSuffix {
			return true
		}
	}
	return false
}
```

In `sourcestate.go`, update external file recognition (around line 1068):
```go
case isPrefixDotFormat(fileInfo.Name(), externalName) || isPrefixDotFormatDotTmpl(fileInfo.Name(), externalName) || isPrefixDotFormatDotJ2(fileInfo.Name(), externalName):
```

Search for any other places where `TemplateSuffix` is used for file matching and add the `JinjaSuffix` equivalent. Check `sourcestate.go` line 1433 (`FormatFromAbsPath`) — this trims `.tmpl` suffix; also trim `.j2`.

- [ ] **Step 4: Update executeTemplate to propagate engine type**

The `executeTemplate` function in `sourcestate.go` (around line 1651) reads a file and calls `ExecuteTemplateData` with no engine type. For `.j2` files (like `.chezmoiignore.j2`), this defaults to the Go engine, which is wrong.

Fix `executeTemplate` to detect the engine type from the file extension:

```go
func (s *SourceState) executeTemplate(templateAbsPath AbsPath) ([]byte, error) {
	data, err := s.system.ReadFile(templateAbsPath)
	if err != nil {
		return nil, err
	}
	var engineType TemplateEngineType
	if strings.HasSuffix(templateAbsPath.String(), JinjaSuffix) {
		engineType = TemplateEngineJinja
	}
	return s.ExecuteTemplateData(ExecuteTemplateDataOptions{
		NameRelPath: templateAbsPath.MustTrimDirPrefix(s.sourceDirAbsPath),
		Data:        data,
		EngineType:  engineType,
	})
}
```

This requires adding the `EngineType` field to `ExecuteTemplateDataOptions` — done in the next step.

- [ ] **Step 5: Run all tests**

Run: `cd /Volumes/Code/claude-workspace-ccl/chezmoi && go test ./internal/chezmoi/ -v -count=1 2>&1 | tail -20`
Expected: All PASS

- [ ] **Step 6: Commit**

```bash
cd /Volumes/Code/claude-workspace-ccl/chezmoi
git add internal/chezmoi/chezmoi.go internal/chezmoi/sourcestate.go internal/chezmoi/format.go
git commit -m "$(cat <<'EOF'
feat: recognize .j2 variants for special chezmoi files

Add .j2 support for .chezmoiignore, .chezmoiremove, .chezmoiexternal,
config templates, and knownPrefixedFiles. executeTemplate detects engine
type from file extension.
EOF
)"
```

---

### Task 7: Wire engine type through ExecuteTemplateData

**Files:**
- Modify: `internal/chezmoi/sourcestate.go`

- [ ] **Step 1: Add EngineType to ExecuteTemplateDataOptions**

```go
type ExecuteTemplateDataOptions struct {
	NameRelPath     RelPath
	DestAbsPath     AbsPath
	Data            []byte
	TemplateOptions TemplateOptions
	ExtraData       map[string]any
	EngineType      TemplateEngineType
}
```

- [ ] **Step 2: Update ExecuteTemplateData to use engine type**

```go
func (s *SourceState) ExecuteTemplateData(options ExecuteTemplateDataOptions) ([]byte, error) {
	templateOptions := options.TemplateOptions
	templateOptions.Funcs = s.templateFuncs
	templateOptions.Options = slices.Clone(s.templateOptions)

	engineType := options.EngineType
	if engineType == "" {
		engineType = TemplateEngineGo
	}

	engine := NewTemplateEngine(engineType)
	if err := engine.Parse(options.NameRelPath.String(), options.Data, templateOptions); err != nil {
		return nil, err
	}

	for name, t := range s.templates {
		if t.EngineType() == engine.EngineType() {
			if err := engine.AddSubTemplate(name, t); err != nil {
				return nil, err
			}
		}
	}

	templateData := s.TemplateData()
	if chezmoiTemplateData, ok := templateData["chezmoi"].(map[string]any); ok {
		chezmoiTemplateData["sourceFile"] = options.NameRelPath.String()
		chezmoiTemplateData["targetFile"] = options.DestAbsPath.String()
	}
	RecursiveMerge(templateData, options.ExtraData)

	result, err := engine.Execute(templateData)
	if errors.Is(err, errReturnEmpty) {
		return nil, nil
	}
	return result, err
}
```

**Note:** `NewTemplateEngine(TemplateEngineJinja)` returns `&JinjaTemplate{}` which doesn't exist yet. For now, keep a compile guard — either stub `JinjaTemplate` or leave the Jinja case commented out until Task 9. The simplest approach: create a minimal stub `jinjatemplate.go` that satisfies the interface but panics on use:

```go
package chezmoi

type JinjaTemplate struct{}

func (t *JinjaTemplate) EngineType() TemplateEngineType { return TemplateEngineJinja }
func (t *JinjaTemplate) Parse(name string, data []byte, options TemplateOptions) error {
	return fmt.Errorf("jinja template engine not yet implemented")
}
func (t *JinjaTemplate) Execute(templateData any) ([]byte, error) {
	return nil, fmt.Errorf("jinja template engine not yet implemented")
}
func (t *JinjaTemplate) AddSubTemplate(name string, engine TemplateEngine) error {
	return fmt.Errorf("jinja template engine not yet implemented")
}
```

- [ ] **Step 3: Update call sites to pass fileAttr.TemplateEngine**

For each `ExecuteTemplateData` call site that has access to `fileAttr`, add `EngineType: fileAttr.TemplateEngine`. There are ~6 call sites in `sourcestate.go` around lines 1927, 1984, 2034, 2145, 2184 that check `fileAttr.Template`. Add the field to each.

- [ ] **Step 4: Update addTemplatesDir to detect engine type**

In `addTemplatesDir` (around line 1620), parse `.j2` files with the Jinja engine:

```go
case fileInfo.Mode().IsRegular():
	contents, err := s.system.ReadFile(templateAbsPath)
	if err != nil {
		return err
	}
	templateRelPath := templateAbsPath.MustTrimDirPrefix(templatesDirAbsPath)
	name := templateRelPath.String()

	var engine TemplateEngine
	if strings.HasSuffix(name, JinjaSuffix) {
		jinja := &JinjaTemplate{}
		if err := jinja.Parse(name, contents, TemplateOptions{
			Funcs:   s.templateFuncs,
			Options: slices.Clone(s.templateOptions),
		}); err != nil {
			return err
		}
		engine = jinja
	} else {
		tmpl, err := ParseTemplate(name, contents, TemplateOptions{
			Funcs:   s.templateFuncs,
			Options: slices.Clone(s.templateOptions),
		})
		if err != nil {
			return err
		}
		engine = tmpl
	}
	s.mutex.Lock()
	s.templates[name] = engine
	s.mutex.Unlock()
	return nil
```

- [ ] **Step 5: Run all tests**

Run: `cd /Volumes/Code/claude-workspace-ccl/chezmoi && go build ./...`
Expected: Clean compile

Run: `cd /Volumes/Code/claude-workspace-ccl/chezmoi && go test ./... -count=1 2>&1 | tail -30`
Expected: All PASS

- [ ] **Step 6: Commit**

```bash
cd /Volumes/Code/claude-workspace-ccl/chezmoi
git add internal/chezmoi/sourcestate.go internal/chezmoi/jinjatemplate.go
git commit -m "$(cat <<'EOF'
feat: wire EngineType through ExecuteTemplateData

Add EngineType to ExecuteTemplateDataOptions. ExecuteTemplateData uses
NewTemplateEngine factory. Call sites pass fileAttr.TemplateEngine.
JinjaTemplate stubbed with not-implemented errors.
EOF
)"
```

---

### Task 8: Implement function adapter layer

**Files:**
- Create: `internal/chezmoi/funcadapter.go`
- Create: `internal/chezmoi/funcadapter_test.go`

**gonja v2 API reference:**
- Filters: `func(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value`
- Globals: Any callable Go function can be passed in the context `map[string]any`
- Values: `exec.AsValue(x)` wraps any Go value, `v.String()`, `v.Integer()` etc. for access

- [ ] **Step 1: Write failing tests**

Create `internal/chezmoi/funcadapter_test.go`:

```go
package chezmoi

import (
	"strings"
	"testing"
	"text/template"

	"github.com/alecthomas/assert/v2"
	"github.com/nikolalohinski/gonja/v2/exec"
)

func TestWrapAsGonjaGlobals(t *testing.T) {
	funcs := template.FuncMap{
		"upper": func(s string) string { return strings.ToUpper(s) },
		"add":   func(a, b int) int { return a + b },
	}

	globals := WrapAsGonjaGlobals(funcs)

	t.Run("all_functions_present", func(t *testing.T) {
		_, ok := globals["upper"]
		assert.True(t, ok)
		_, ok = globals["add"]
		assert.True(t, ok)
	})
}

func TestWrapAsGonjaFilters(t *testing.T) {
	funcs := template.FuncMap{
		"upper": func(s string) string { return strings.ToUpper(s) },
	}

	filters := WrapAsGonjaFilters(funcs)

	t.Run("single_arg_becomes_filter", func(t *testing.T) {
		fn, ok := filters["upper"]
		assert.True(t, ok)
		assert.NotEqual(t, nil, fn)
	})
}

func TestGonjaGlobalFunctionExecution(t *testing.T) {
	// End-to-end: register a function via adapter, execute via gonja template
	funcs := template.FuncMap{
		"greet": func(name string) string { return "Hello, " + name + "!" },
	}

	globals := WrapAsGonjaGlobals(funcs)

	// Create a JinjaTemplate with the adapted functions
	engine := &JinjaTemplate{}
	options := TemplateOptions{
		Funcs: template.FuncMap(funcs),
	}
	err := engine.Parse("test", []byte(`{{ greet("World") }}`), options)
	assert.NoError(t, err)

	result, err := engine.Execute(map[string]any{})
	assert.NoError(t, err)
	assert.Equal(t, "Hello, World!", string(result))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Volumes/Code/claude-workspace-ccl/chezmoi && go test ./internal/chezmoi/ -run "TestWrapAsGonja|TestGonjaGlobal" -v`
Expected: FAIL

- [ ] **Step 3: Implement the adapter**

Create `internal/chezmoi/funcadapter.go`:

```go
package chezmoi

import (
	"reflect"
	"text/template"

	"github.com/nikolalohinski/gonja/v2/exec"
)

// WrapAsGonjaGlobals converts a template.FuncMap into gonja-compatible
// global functions. Gonja can call any Go function placed in the context,
// so for globals we just return the map as-is.
func WrapAsGonjaGlobals(funcs template.FuncMap) map[string]any {
	globals := make(map[string]any, len(funcs))
	for name, fn := range funcs {
		globals[name] = fn
	}
	return globals
}

// WrapAsGonjaFilters converts functions with a "transform" signature
// (first arg is the piped input) into gonja filter functions.
// Only functions whose first parameter can serve as piped input are included.
func WrapAsGonjaFilters(funcs template.FuncMap) map[string]exec.FilterFunction {
	filters := make(map[string]exec.FilterFunction)
	for name, fn := range funcs {
		fnVal := reflect.ValueOf(fn)
		fnType := fnVal.Type()
		// Only wrap functions with at least 1 parameter (the piped input)
		if fnType.NumIn() < 1 {
			continue
		}
		// Capture for closure
		name, fnVal, fnType := name, fnVal, fnType
		filters[name] = func(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
			args := make([]reflect.Value, fnType.NumIn())
			// First arg is the piped input
			args[0] = convertToReflectValue(in, fnType.In(0))
			// Remaining args from params
			for i := 1; i < fnType.NumIn(); i++ {
				if i-1 < len(params.Args) {
					args[i] = convertToReflectValue(params.Args[i-1], fnType.In(i))
				} else {
					args[i] = reflect.Zero(fnType.In(i))
				}
			}
			results := fnVal.Call(args)
			if len(results) == 0 {
				return exec.AsValue(nil)
			}
			// If last return is error, check it
			if fnType.NumOut() == 2 && fnType.Out(1).Implements(reflect.TypeOf((*error)(nil)).Elem()) {
				if !results[1].IsNil() {
					return exec.AsValue(results[1].Interface().(error))
				}
			}
			return exec.AsValue(results[0].Interface())
		}
	}
	return filters
}

// convertToReflectValue converts a gonja *exec.Value to a reflect.Value
// of the target type.
func convertToReflectValue(v *exec.Value, targetType reflect.Type) reflect.Value {
	switch targetType.Kind() {
	case reflect.String:
		return reflect.ValueOf(v.String())
	case reflect.Int, reflect.Int64:
		return reflect.ValueOf(int(v.Integer())).Convert(targetType)
	case reflect.Float64:
		return reflect.ValueOf(v.Float())
	case reflect.Bool:
		return reflect.ValueOf(v.Bool())
	default:
		// For complex types (maps, slices, etc.), try direct interface
		return reflect.ValueOf(v.Interface())
	}
}
```

**Note:** The exact `exec.Value` methods and `exec.FilterFunction` type signature must be verified against gonja v2's actual API. The structure above follows the research findings. Adjust based on what compiles.

- [ ] **Step 4: Run tests**

Run: `cd /Volumes/Code/claude-workspace-ccl/chezmoi && go test ./internal/chezmoi/ -run "TestWrapAsGonja|TestGonjaGlobal" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd /Volumes/Code/claude-workspace-ccl/chezmoi
git add internal/chezmoi/funcadapter.go internal/chezmoi/funcadapter_test.go
git commit -m "$(cat <<'EOF'
feat: add function adapter for gonja template engine

WrapAsGonjaGlobals passes functions through as Go callables.
WrapAsGonjaFilters wraps transform-style functions as gonja filters
using reflection for type conversion.
EOF
)"
```

---

### Task 9: Implement JinjaTemplate (full)

**Files:**
- Modify: `internal/chezmoi/jinjatemplate.go` — replace stub with full implementation
- Create: `internal/chezmoi/jinjatemplate_test.go`

**gonja v2 API summary:**
- Parse: `gonja.FromString(source)` returns `(*exec.Template, error)`
- Execute: `template.ExecuteToString(context)` or `template.Execute(writer, context)`
- Context: `exec.NewContext(map[string]any{...})`
- Filters: `gonja.DefaultEnvironment.Filters.Register(name, filterFunc)`

- [ ] **Step 1: Write failing tests**

Create `internal/chezmoi/jinjatemplate_test.go`:

```go
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

func TestJinjaTemplateDirective(t *testing.T) {
	engine := &JinjaTemplate{}
	input := "{# chezmoi:template:line-ending=crlf #}\nline1\nline2\n"
	err := engine.Parse("test", []byte(input), TemplateOptions{})
	assert.NoError(t, err)

	result, err := engine.Execute(nil)
	assert.NoError(t, err)
	assert.Equal(t, "line1\r\nline2\r\n", string(result))
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

func TestJinjaTemplateGoSpecificDirectiveIgnored(t *testing.T) {
	engine := &JinjaTemplate{}
	input := "{# chezmoi:template:left-delimiter=[[ right-delimiter=]] #}\nhello"
	err := engine.Parse("test", []byte(input), TemplateOptions{})
	assert.NoError(t, err)
	result, err := engine.Execute(nil)
	assert.NoError(t, err)
	assert.Equal(t, "hello", string(result))
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Volumes/Code/claude-workspace-ccl/chezmoi && go test ./internal/chezmoi/ -run TestJinjaTemplate -v`
Expected: FAIL (stub returns errors)

- [ ] **Step 3: Implement JinjaTemplate**

Replace the stub in `internal/chezmoi/jinjatemplate.go` with the full implementation:

```go
package chezmoi

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/mitchellh/copystructure"
	"github.com/nikolalohinski/gonja/v2"
	"github.com/nikolalohinski/gonja/v2/exec"
)

// JinjaTemplate implements TemplateEngine using gonja v2 (Jinja2 for Go).
type JinjaTemplate struct {
	name     string
	env      *gonja.Environment  // per-template environment (avoids global mutation)
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

	// Build globals from template functions
	t.globals = make(map[string]any)
	if options.Funcs != nil {
		for k, v := range WrapAsGonjaGlobals(options.Funcs) {
			t.globals[k] = v
		}
	}

	// Create a per-template environment to avoid global filter mutation.
	// This prevents test interference and cross-template filter leakage.
	env := gonja.NewEnvironment(gonja.DefaultConfig, gonja.DefaultLoader)

	// Copy default filters into the new environment
	// (gonja.NewEnvironment may already include defaults — verify)

	// Register custom filters for transform-style functions
	if options.Funcs != nil {
		filters := WrapAsGonjaFilters(options.Funcs)
		for filterName, filterFunc := range filters {
			_ = env.Filters.Register(filterName, filterFunc)
		}
	}

	tmpl, err := env.FromBytes(contents)
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

	// Build context from data + globals
	contextMap := make(map[string]any)
	// Add globals first
	for k, v := range t.globals {
		contextMap[k] = v
	}
	// Add template data (overrides globals on conflict)
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
// Registers the sub-template's source in the gonja environment's loader
// so that {% include "name" %} and {% extends "name" %} resolve correctly.
func (t *JinjaTemplate) AddSubTemplate(name string, engine TemplateEngine) error {
	jinjaEngine, ok := engine.(*JinjaTemplate)
	if !ok {
		return fmt.Errorf("cannot add %s template %q to Jinja template engine", engine.EngineType(), name)
	}
	// Store the sub-template's parsed template in our environment.
	// gonja v2's environment has a template cache or loader that can
	// be populated. The exact mechanism depends on the gonja API:
	//
	// Option A: If gonja supports a MemoryLoader or similar:
	//   t.env.Loader.Set(name, jinjaEngine.source)
	//
	// Option B: If gonja uses a template cache:
	//   t.env.Cache[name] = jinjaEngine.template
	//
	// The implementer must check gonja v2's loader interface and use
	// the appropriate registration method. If neither works, implement
	// a custom loader that wraps a map[string][]byte of template sources.
	//
	// For the MVP, if gonja's loader doesn't support dynamic registration,
	// store sub-template sources in JinjaTemplate and implement a custom
	// loaders.Loader that serves them.
	_ = jinjaEngine // used by the chosen registration approach
	return nil
}
```

**Important notes:**
- The gonja v2 API may differ from what's shown. Adjust based on what compiles: check `gonja.FromBytes` vs `gonja.FromString`, `exec.Template` vs other types, `template.Execute` signature (may take `io.Writer` + context or return string).
- Filter registration on `gonja.DefaultEnvironment` is global and may cause test interference. If so, create a per-template environment instead.
- The `{{ greet("World") }}` syntax requires that gonja supports calling Go functions from context. Verify this works.

- [ ] **Step 4: Iterate until tests pass**

Run: `cd /Volumes/Code/claude-workspace-ccl/chezmoi && go test ./internal/chezmoi/ -run TestJinjaTemplate -v`
Expected: All PASS

Fix any compilation or runtime issues. The gonja API research provides guidance but the exact types may need adjustment.

- [ ] **Step 5: Run all tests (regression check)**

Run: `cd /Volumes/Code/claude-workspace-ccl/chezmoi && go test ./... -count=1 2>&1 | tail -30`
Expected: All PASS

- [ ] **Step 6: Commit**

```bash
cd /Volumes/Code/claude-workspace-ccl/chezmoi
git add internal/chezmoi/jinjatemplate.go internal/chezmoi/jinjatemplate_test.go
git commit -m "$(cat <<'EOF'
feat: implement JinjaTemplate engine using gonja v2

Full JinjaTemplate implementation supporting variables, conditionals,
loops, nested data, chezmoi directives, function calls via context
globals, deep-copy safety, and line ending normalization.
EOF
)"
```

---

### Task 10: Contract tests — behavioral parity

**Files:**
- Create: `internal/chezmoi/templateengine_contract_test.go`

- [ ] **Step 1: Write contract test suite**

Create `internal/chezmoi/templateengine_contract_test.go`:

```go
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
```

- [ ] **Step 2: Run contract tests**

Run: `cd /Volumes/Code/claude-workspace-ccl/chezmoi && go test ./internal/chezmoi/ -run TestContract -v`
Expected: All PASS

- [ ] **Step 3: Commit**

```bash
cd /Volumes/Code/claude-workspace-ccl/chezmoi
git add internal/chezmoi/templateengine_contract_test.go
git commit -m "$(cat <<'EOF'
test: add contract tests for TemplateEngine behavioral parity

Shared test suite verifying GoTemplate and JinjaTemplate produce
identical output for variables, nested data, functions, deep-copy,
and post-processing.
EOF
)"
```

---

### Task 11: Jinja-specific feature tests

**Files:**
- Modify: `internal/chezmoi/jinjatemplate_test.go`

- [ ] **Step 1: Add Jinja-specific tests**

Add to `internal/chezmoi/jinjatemplate_test.go`:

```go
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
	err := engine.Parse("test", []byte(`{{ "hello " + name }}`), TemplateOptions{})
	assert.NoError(t, err)
	result, err := engine.Execute(map[string]any{"name": "world"})
	assert.NoError(t, err)
	assert.Equal(t, "hello world", string(result))
}
```

- [ ] **Step 2: Run tests**

Run: `cd /Volumes/Code/claude-workspace-ccl/chezmoi && go test ./internal/chezmoi/ -run TestJinjaTemplate -v`
Expected: All PASS

- [ ] **Step 3: Commit**

```bash
cd /Volumes/Code/claude-workspace-ccl/chezmoi
git add internal/chezmoi/jinjatemplate_test.go
git commit -m "$(cat <<'EOF'
test: add Jinja-specific feature tests

Test filter chaining, default filter, and string expression
concatenation — Jinja features not available in Go templates.
EOF
)"
```

---

### Task 12: Integration test — end-to-end .j2 file

**Files:**
- Create: `internal/cmd/testdata/scripts/jinjaTemplate.txtar`

Chezmoi uses `.txtar` test scripts. Each script contains commands and embedded file data.

- [ ] **Step 1: Create .txtar integration test**

Create `internal/cmd/testdata/scripts/jinjaTemplate.txtar`:

```
# Test that .j2 files are processed with the Jinja template engine
mksourcedir

exec chezmoi apply --force
cmp $HOME/.bashrc golden/.bashrc

-- home/user/.local/share/chezmoi/dot_bashrc.j2 --
# Generated by chezmoi
{% if chezmoi.os == "linux" %}
export PATH="/usr/local/bin:$PATH"
{% else %}
export PATH="/opt/homebrew/bin:$PATH"
{% endif %}
-- golden/.bashrc --
# Generated by chezmoi
export PATH="/opt/homebrew/bin:$PATH"
```

**Note:** The golden output depends on the test runner's OS. Use a simpler template if OS-dependent output is problematic:

```
# Test that .j2 files are processed with the Jinja template engine
mksourcedir

exec chezmoi apply --force
cmp $HOME/.config golden/.config

-- home/user/.local/share/chezmoi/dot_config.j2 --
name={{ name }}
-- home/user/.config/chezmoi/chezmoi.toml --
[data]
name = "test"
-- golden/.config --
name=test
```

Adjust the `.txtar` format to match the exact conventions used in the existing test scripts. Look at existing scripts for the correct home directory structure and `mksourcedir` usage.

- [ ] **Step 2: Run the integration test**

Run: `cd /Volumes/Code/claude-workspace-ccl/chezmoi && go test ./internal/cmd/ -run TestScript/jinjaTemplate -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
cd /Volumes/Code/claude-workspace-ccl/chezmoi
git add internal/cmd/testdata/scripts/jinjaTemplate.txtar
git commit -m "$(cat <<'EOF'
test: add integration test for .j2 template execution

End-to-end txtar test verifying .j2 source files are processed
through the Jinja engine with template data access.
EOF
)"
```

---

### Task 13: Full test suite verification

- [ ] **Step 1: Build**

Run: `cd /Volumes/Code/claude-workspace-ccl/chezmoi && go build ./...`
Expected: Clean compile

- [ ] **Step 2: Run full test suite**

Run: `cd /Volumes/Code/claude-workspace-ccl/chezmoi && go test ./... -count=1 2>&1 | tail -50`
Expected: All PASS

- [ ] **Step 3: Run vet**

Run: `cd /Volumes/Code/claude-workspace-ccl/chezmoi && go vet ./...`
Expected: No issues

- [ ] **Step 4: Fix any failures and commit**

If any tests fail, diagnose and fix the root cause.

```bash
cd /Volumes/Code/claude-workspace-ccl/chezmoi
git add -A
git commit -m "$(cat <<'EOF'
fix: resolve issues from full test suite verification
EOF
)"
```
