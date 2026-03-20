# Jinja Dotfiles Validation Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Create txtar integration tests and unit tests that validate the Jinja template engine against real-world dotfiles patterns.

**Architecture:** Four test files (3 txtar + 1 unit test), each targeting a distinct engine code path. Tests use user-defined `[data]` fields for deterministic behavior. Struct field access (the riskiest path) is validated via unit tests with mock functions.

**Tech Stack:** Go testscript (txtar format), go test, gonja v2, chezmoi funcadapter

**Spec:** `docs/superpowers/specs/2026-03-19-jinja-dotfiles-validation-design.md`

---

### Task 1: Tier 1 — Boolean Operators and Bracket Index Access

**Files:**
- Create: `internal/cmd/testdata/scripts/jinjaTemplateBooleans.txtar`

This test validates Jinja native operators (`==`, `and`, `or`, `not`), bracket index access on maps, compound boolean expressions, and truthiness testing. All data is user-defined via `[data]` for deterministic cross-platform behavior.

- [ ] **Step 1: Verify `in` operator behavior on strings in gonja**

Before writing the test, we need to know if gonja's `in` supports string containment. Run this quick check:

```bash
cd /Volumes/Code/claude-workspace-ccl/chezmoi
TMPDIR=/private/tmp/claude-501 go test ./internal/chezmoi/ -run TestJinjaStringIn -count 1 -v
```

This will fail because the test doesn't exist yet. Create a temporary test in `internal/chezmoi/jinjatemplate_test.go`:

```go
func TestJinjaStringIn(t *testing.T) {
	engine := &JinjaTemplate{}
	err := engine.Parse("test", []byte(`{% if "micro" in kernel %}yes{% else %}no{% endif %}`), TemplateOptions{})
	assert.NoError(t, err)
	result, err := engine.Execute(map[string]any{"kernel": "5.15.0-microsoft-standard"})
	assert.NoError(t, err)
	// If gonja supports string containment, result is "yes"
	// If not, result is "no" and we need to use contains() instead
	t.Logf("in operator on string result: %q", string(result))
}
```

Run it and check the output. If `in` returns "yes", use native `in` in the txtar test. If "no", use `contains("substring", str)` as fallback. Remove this temporary test after checking.

- [ ] **Step 2: Write the txtar test**

Create `internal/cmd/testdata/scripts/jinjaTemplateBooleans.txtar`:

```
# Test Jinja native boolean operators and bracket index access on maps

exec chezmoi apply --force

cmp $HOME/.platform   golden/.platform
cmp $HOME/.wsl-check  golden/.wsl-check
cmp $HOME/.os-info    golden/.os-info

-- home/user/.config/chezmoi/chezmoi.toml --
[data]
    platform = "linux"
    is_wsl = true
    is_container = false
    codespaces = false
    kernel_release = "5.15.0-microsoft-standard"

[data.os_info]
    id = "ubuntu"
    id_like = "debian"
    version = "22.04"

-- home/user/.local/share/chezmoi/dot_platform.j2 --
{%- if platform == "linux" -%}
PLATFORM=linux
{%- elif platform == "darwin" -%}
PLATFORM=macos
{%- else -%}
PLATFORM=other
{%- endif %}
-- golden/.platform --
PLATFORM=linux

-- home/user/.local/share/chezmoi/dot_wsl-check.j2 --
{%- if platform == "linux" and not is_container -%}
{%- if is_wsl -%}
ENV=wsl
{%- else -%}
ENV=native
{%- endif -%}
{%- else -%}
ENV=other
{%- endif %}
-- golden/.wsl-check --
ENV=wsl

-- home/user/.local/share/chezmoi/dot_os-info.j2 --
id={{ os_info["id"] }}
id_like={{ os_info["id_like"] }}
codespaces={{ codespaces }}
-- golden/.os-info --
id=ubuntu
id_like=debian
codespaces=False
```

**Note:** The `codespaces=False` golden value assumes gonja renders boolean `false` as `False` (Python-style). If it renders as `false` or empty string, adjust the golden file accordingly.

- [ ] **Step 3: Run the test**

```bash
cd /Volumes/Code/claude-workspace-ccl/chezmoi
TMPDIR=/private/tmp/claude-501 go test ./internal/cmd/ -run 'TestScript/jinjaTemplateBooleans' -count 1 -v
```

Expected: PASS. If the golden files don't match (whitespace, boolean rendering), examine the diff output and adjust the golden files or whitespace control tags (`{%- -%}`) until it passes.

- [ ] **Step 4: Commit**

```bash
git add internal/cmd/testdata/scripts/jinjaTemplateBooleans.txtar
git commit -m "test: add Jinja boolean operators and bracket index integration test

Validates native ==, and, or, not operators, compound boolean
expressions, bracket index access on maps, and boolean truthiness
rendering through chezmoi apply with .j2 templates."
```

---

### Task 2: Tier 2 — Sub-template Includes with Data Context

**Files:**
- Create: `internal/cmd/testdata/scripts/jinjaTemplateIncludeData.txtar`

This test validates that `{% include %}` passes template data context into included partials that contain conditionals and nested data access. This is net-new over the existing `jinjaTemplateInclude.txtar` which only tests static partials.

- [ ] **Step 1: Write the txtar test**

Create `internal/cmd/testdata/scripts/jinjaTemplateIncludeData.txtar`:

```
# Test that included .j2 sub-templates receive template data context

exec chezmoi apply --force

cmp $HOME/.editor-config golden/.editor-config

-- home/user/.config/chezmoi/chezmoi.toml --
[data]
    editor = "vim"
    use_lsp = true

[data.settings]
    theme = "dark"
    tab_size = 4

-- home/user/.local/share/chezmoi/.chezmoitemplates/config/editor.j2 --
editor={{ editor }}
{%- if use_lsp %}
lsp=enabled
{%- else %}
lsp=disabled
{%- endif %}
-- home/user/.local/share/chezmoi/.chezmoitemplates/config/theme.j2 --
theme={{ settings.theme }}
tab_size={{ settings.tab_size }}
-- home/user/.local/share/chezmoi/dot_editor-config.j2 --
# Editor Configuration
{% include "config/editor.j2" %}
{% include "config/theme.j2" %}
-- golden/.editor-config --
# Editor Configuration
editor=vim
lsp=enabled
theme=dark
tab_size=4
```

- [ ] **Step 2: Run the test**

```bash
cd /Volumes/Code/claude-workspace-ccl/chezmoi
TMPDIR=/private/tmp/claude-501 go test ./internal/cmd/ -run 'TestScript/jinjaTemplateIncludeData' -count 1 -v
```

Expected: PASS. If whitespace doesn't match, adjust the template whitespace control or golden file. The key thing to verify is that `editor`, `use_lsp`, and `settings.theme` are all accessible inside the included partials.

- [ ] **Step 3: Commit**

```bash
git add internal/cmd/testdata/scripts/jinjaTemplateIncludeData.txtar
git commit -m "test: add Jinja include with data context integration test

Validates that {% include %} passes template data into sub-templates
that use conditionals and nested map access, extending coverage beyond
the static-partial test in jinjaTemplateInclude.txtar."
```

---

### Task 3: Tier 3a — Iteration over Arrays of Maps (txtar)

**Files:**
- Create: `internal/cmd/testdata/scripts/jinjaTemplateIteration.txtar`

This test validates `for` loops over TOML arrays containing nested maps, with field access inside the loop body. This exercises gonja's iteration over `[]any` containing `map[string]any` values — complementary to the struct reflection test in Task 4.

- [ ] **Step 1: Write the txtar test**

Create `internal/cmd/testdata/scripts/jinjaTemplateIteration.txtar`:

```
# Test Jinja for loops over arrays of maps with field access

exec chezmoi apply --force

cmp $HOME/.ssh-keys   golden/.ssh-keys
cmp $HOME/.packages   golden/.packages

-- home/user/.config/chezmoi/chezmoi.toml --
[[data.ssh_keys]]
    type = "ed25519"
    comment = "alice@work"

[[data.ssh_keys]]
    type = "rsa"
    comment = "alice@home"

[[data.packages]]
    name = "git"
    required = true

[[data.packages]]
    name = "curl"
    required = true

[[data.packages]]
    name = "neofetch"
    required = false

-- home/user/.local/share/chezmoi/dot_ssh-keys.j2 --
# SSH Keys
{%- for key in ssh_keys %}
{{ key.type }} {{ key.comment }}
{%- endfor %}
-- golden/.ssh-keys --
# SSH Keys
ed25519 alice@work
rsa alice@home

-- home/user/.local/share/chezmoi/dot_packages.j2 --
# Required packages
{%- for pkg in packages -%}
{%- if pkg.required %}
{{ pkg.name }}
{%- endif -%}
{%- endfor %}
-- golden/.packages --
# Required packages
git
curl
```

- [ ] **Step 2: Run the test**

```bash
cd /Volumes/Code/claude-workspace-ccl/chezmoi
TMPDIR=/private/tmp/claude-501 go test ./internal/cmd/ -run 'TestScript/jinjaTemplateIteration' -count 1 -v
```

Expected: PASS. The TOML `[[data.ssh_keys]]` array-of-tables syntax creates `[]any` containing `map[string]any` entries — verify gonja can iterate over these and access fields with dot notation.

- [ ] **Step 3: Commit**

```bash
git add internal/cmd/testdata/scripts/jinjaTemplateIteration.txtar
git commit -m "test: add Jinja iteration over arrays of maps integration test

Validates for loops over TOML array-of-tables data with field access
and conditional filtering inside loop bodies."
```

---

### Task 4: Tier 3b — Struct Field Access via Reflection (Unit Test)

**Files:**
- Modify: `internal/chezmoi/jinjatemplate_test.go`

This is the highest-value test. It validates that Go functions returning structs work correctly through the Jinja engine — the `convertToReflectValue` default path in `funcadapter.go` and gonja's struct reflection. This mimics `gitHubKeys` and `rbw` patterns from the dotfiles.

- [ ] **Step 1: Write the struct field access test**

Add to `internal/chezmoi/jinjatemplate_test.go`:

```go
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
```

- [ ] **Step 2: Run the test**

```bash
cd /Volumes/Code/claude-workspace-ccl/chezmoi
TMPDIR=/private/tmp/claude-501 go test ./internal/chezmoi/ -run 'TestJinjaTemplateStructFieldAccess' -count 1 -v
```

Expected: PASS for all three sub-tests. If any fail:
- `for_loop_with_struct_field_access`: gonja cannot iterate over `[]SSHKey` or cannot access `.Key` on struct — this means the funcadapter's `exec.AsValue()` isn't wrapping structs correctly
- `struct_field_access_on_function_return`: gonja cannot resolve `.Notes` on a struct returned by a function call
- `struct_slice_index_and_field_access`: gonja cannot handle `Fields[0].Value` bracket+dot chaining

If any subtests fail, the failure indicates a real limitation in the funcadapter or gonja's reflection layer that needs to be documented or fixed.

- [ ] **Step 3: Run all Jinja tests to verify no regressions**

```bash
cd /Volumes/Code/claude-workspace-ccl/chezmoi
TMPDIR=/private/tmp/claude-501 go test ./internal/chezmoi/ -run 'TestJinja|TestContract' -count 1 -v
TMPDIR=/private/tmp/claude-501 go test ./internal/cmd/ -run 'TestScript/jinjaTemplate' -count 1 -v
```

Expected: All tests PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/chezmoi/jinjatemplate_test.go
git commit -m "test: add Jinja struct field access unit tests

Validates that Go functions returning structs work through the Jinja
engine: for loops over struct slices, field access on function return
values, and bracket index + dot chaining on nested struct fields.
Mimics gitHubKeys and rbw patterns from real dotfiles."
```

---

### Task 5: Final Validation

- [ ] **Step 1: Run all Jinja tests together**

```bash
cd /Volumes/Code/claude-workspace-ccl/chezmoi
TMPDIR=/private/tmp/claude-501 go test ./internal/chezmoi/ -run 'TestJinja|TestContract' -count 1 -v
TMPDIR=/private/tmp/claude-501 go test ./internal/cmd/ -run 'TestScript/jinjaTemplate' -count 1 -v
```

Expected: All tests PASS — the full list should include:
- `jinjaTemplate` (original)
- `jinjaTemplateBooleans` (new — Task 1)
- `jinjaTemplateControlFlow`
- `jinjaTemplateErrors`
- `jinjaTemplateFilters`
- `jinjaTemplateFuncs`
- `jinjaTemplateInclude`
- `jinjaTemplateIncludeData` (new — Task 2)
- `jinjaTemplateIteration` (new — Task 3)
- `jinjaTemplateMixed`
- Plus all `TestJinja*` and `TestContract*` unit tests

- [ ] **Step 2: Verify test count**

Confirm we now have 10 txtar integration tests and the full unit test suite. The new tests add coverage for: boolean operators, bracket index access, include with data context, array iteration with maps, and struct field access via reflection.
