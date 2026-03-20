# Jinja Engine Validation via Dotfiles-Derived Test Fixtures

**Date:** 2026-03-19
**Goal:** Dogfood the Jinja template engine by creating txtar integration tests that mirror real-world dotfiles template patterns, validating engine code paths in CI.

## Context

The Jinja template engine (`JinjaTemplate`) supports variable interpolation, control flow, filter syntax, function calls via `funcadapter.go`, and sub-template includes via `mutableLoader`. These capabilities need validation against real template patterns, not just synthetic unit tests.

The test fixtures are derived from [tylerbutler/dotfiles](https://github.com/tylerbutler/dotfiles), which contains 41 `.tmpl` files. Rather than converting the actual dotfiles repo, we create txtar integration tests in chezmoi that embed representative patterns from those templates.

### Existing Test Coverage

Before designing new tests, here is what the existing `jinja*.txtar` tests already cover:

| Code Path | Existing Test |
|---|---|
| Variable interpolation | `jinjaTemplate.txtar` |
| `if`/`elif`/`else` on `chezmoi.os` | `jinjaTemplateControlFlow.txtar` |
| `for` loops with `loop.index` | `jinjaTemplateControlFlow.txtar` |
| `{% include %}` with nested paths | `jinjaTemplateInclude.txtar` |
| Filter pipeline (`\| trim \| upper`) | `jinjaTemplateFilters.txtar` |
| `fromJson`, `toJson`, `joinPath` | `jinjaTemplateFuncs.txtar` |
| Mixed `.tmpl` + `.j2` coexistence | `jinjaTemplateMixed.txtar` |
| Parse error reporting | `jinjaTemplateErrors.txtar` |

**Net-new coverage needed:** Native boolean operators (`and`, `or`, `not`, `==`, `in`), bracket index access on maps, compound boolean expressions, struct field access via reflection, `for` loops over function return values.

## Approach: Coverage-Optimized Tiers

Each tier targets a distinct engine code path. Tests are txtar scripts in `internal/cmd/testdata/scripts/`, following existing conventions (no `Dotfiles` infix — use `jinjaTemplate*.txtar` naming).

### Tier 1: Native Boolean Operators and Bracket Index Access

**Engine paths exercised:** Jinja native operators (`==`, `!=`, `in`, `and`, `or`, `not`), compound boolean expressions, bracket index access on maps (`data["key"]`).

**Important constraint:** `chezmoi.os`, `chezmoi.kernel`, and `chezmoi.osRelease` are auto-populated at runtime from the actual OS — they **cannot** be overridden via `[data]` in chezmoi.toml. To test Jinja engine operators deterministically, we use user-defined `[data]` fields that mimic the same access patterns.

**Source patterns (from dotfiles):**

| Go template | Jinja equivalent (using user data) |
|---|---|
| `{{ if eq .chezmoi.os "linux" }}` | `{% if platform == "linux" %}` |
| `{{ if not (or (contains "X" .y) ...) }}` | `{% if not ("X" in kernel_release or ...) %}` |
| `{{ if .codespaces }}` | `{% if codespaces %}` |
| `{{ index .chezmoi.osRelease "idLike" }}` | `{{ os_info["id_like"] }}` |

**Note on `in` for string containment:** Jinja2's `in` operator supports substring containment on strings. gonja v2's behavior must be verified — if it only supports membership testing on lists/dicts, the fallback is `{% if contains("substring", str) %}`.

**txtar test:** `jinjaTemplateBooleans.txtar`

Embeds 2-3 fixture `.j2` files using user-defined `[data]` that cover:
- Simple equality (`{% if platform == "linux" %}`)
- Compound `and`/`or`/`not` expressions (`{% if not (is_wsl or is_container) %}`)
- Bracket index access on a map (`{{ os_info["id_like"] }}`)
- Truthiness test on a boolean data flag (`{% if codespaces %}`)
- Nested conditionals (`{% if platform == "linux" %}{% if not is_wsl %}...{% endif %}{% endif %}`)

All data values are controlled by `[data]` in the test's `chezmoi.toml`, making the test deterministic across all platforms.

### Tier 2: Sub-template Includes with Variable Interpolation

**Engine paths exercised:** `{% include %}` with variable interpolation inside included partials (the existing `jinjaTemplateInclude.txtar` only tests static partials and simple variable access — this tier adds partials that use conditionals and access template data).

**Source patterns (from dotfiles):**

```
{{- template "lsd/config.yaml" . -}}   →  {%- include "lsd/config.yaml.j2" -%}
{{- template "rbw/config.json" . -}}    →  {%- include "rbw/config.json.j2" -%}
```

**Net-new vs existing `jinjaTemplateInclude.txtar`:** The existing test includes a static header and a simple `user={{ name }}` partial. This tier tests:
- A partial that contains Jinja conditionals (`{% if ... %}`)
- A partial that accesses nested data (`{{ settings.editor }}`)
- Verifying that template data context is passed through to included templates

**txtar test:** `jinjaTemplateIncludeData.txtar`

Embeds:
- `.chezmoitemplates/config/editor.j2` — a partial with conditional logic based on template data
- `.chezmoitemplates/config/paths.j2` — a partial accessing nested map data
- A `.j2` file that includes both partials and also has its own content

### Tier 3: Struct Field Access and Iteration

**Engine paths exercised:** `convertToReflectValue` default path with Go struct types returned by functions, `for` loops over function return values, field access on structs via gonja reflection, bracket index access on struct slice fields.

**Source patterns (from dotfiles):**

```
{{ range gitHubKeys "tylerbutler" }}{{ .Key }}{{ end }}
  → {% for key in gitHubKeys("tylerbutler") %}{{ key.Key }}{% endfor %}

{{ (rbw "tylerbu@sshKey").notes }}
  → {{ rbw("tylerbu@sshKey").notes }}

{{ (index (rbw "tylerbu@sshKey").fields 0).value }}
  → {{ rbw("tylerbu@sshKey").fields[0].value }}
```

**Implementation:** Since `rbw` and `gitHubKeys` require external services and txtar tests use the real function registry (no mock injection), this tier uses two complementary tests:

**Go unit test:** `TestJinjaTemplateStructFieldAccess` in `jinjatemplate_test.go`
- Registers mock functions via `TemplateOptions.Funcs` that return:
  - `[]struct{ Key string }` — mimics `gitHubKeys` return
  - `struct{ Notes string; Fields []struct{ Value string } }` — mimics `rbw` return
- Parses Jinja templates using `for` loops, field access, and bracket index
- Validates the funcadapter correctly passes struct values through gonja's reflection

**txtar test:** `jinjaTemplateIteration.txtar`
- Tests `for` loops over TOML arrays with nested map access from `[data]`
- This exercises gonja's iteration and map field access code paths (complementary to the unit test's struct reflection path)
- Includes: iterating over a list of maps, accessing fields within each, and building output

### Skipped Files (2)

These remain as `.tmpl` to validate mixed-engine coexistence:

- **`.chezmoi.toml.tmpl`** — Uses `$var := expr` with reassignment inside `if` blocks (Go-specific scoping), `| not | not` coercion pipe, `| quote` filter. Jinja's `{% set %}` is block-scoped, making direct translation incorrect.
- **`_scripts/run_onchange_install-brewfile.sh.tmpl`** — Uses `include "Brewfile" | sha256sum` where `include` is a chezmoi template function. As a global function name, `include` conflicts with Jinja's `{% include %}` control structure keyword.

## Test File Summary

| Test File | Tier | Net-New Coverage |
|---|---|---|
| `jinjaTemplateBooleans.txtar` | 1 | Native `==`, `and`, `or`, `not`, bracket index, compound expressions |
| `jinjaTemplateIncludeData.txtar` | 2 | Include with conditionals and nested data in partials |
| `jinjaTemplateIteration.txtar` | 3 | `for` over arrays of maps, map field access in loops |
| `TestJinjaTemplateStructFieldAccess` (unit) | 3 | Struct field access via reflection, mock functions returning structs |

## Success Criteria

1. All txtar tests pass on Linux, macOS, and Windows CI
2. Tier 3 unit test validates struct field access works through funcadapter
3. No regressions in existing Jinja or Go template tests
4. Mixed `.tmpl` + `.j2` coexistence validated (already covered by `jinjaTemplateMixed.txtar`)
5. `in` operator string containment behavior in gonja is verified (Tier 1)
