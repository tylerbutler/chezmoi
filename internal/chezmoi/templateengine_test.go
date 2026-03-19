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
