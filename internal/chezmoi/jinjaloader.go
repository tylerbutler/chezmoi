package chezmoi

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/nikolalohinski/gonja/v2/loaders"
)

// mutableLoader is a loaders.Loader that allows templates to be added after
// creation. This enables sub-template registration via AddSubTemplate to work
// with gonja's {% include %} directive at execution time.
type mutableLoader struct {
	root    string
	content map[string]string
}

var _ loaders.Loader = (*mutableLoader)(nil)

// newMutableLoader creates a loader with an initial root template.
func newMutableLoader(rootID, rootContent string) *mutableLoader {
	return &mutableLoader{
		root: filepath.Dir(rootID),
		content: map[string]string{
			rootID: rootContent,
		},
	}
}

// Add registers a sub-template source that can be resolved by {% include %}.
func (m *mutableLoader) Add(name, content string) {
	key := "/" + name
	m.content[key] = content
}

func (m *mutableLoader) Read(path string) (io.Reader, error) {
	resolved, err := m.Resolve(path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve name '%s': %w", path, err)
	}
	data, ok := m.content[resolved]
	if !ok {
		return nil, fmt.Errorf("unknown path: '%s'", resolved)
	}
	return strings.NewReader(data), nil
}

func (m *mutableLoader) Resolve(path string) (string, error) {
	if strings.HasPrefix(path, "/") {
		if _, ok := m.content[path]; ok {
			return path, nil
		}
		return "", fmt.Errorf("unknown resolved path: '%s'", path)
	}
	resolved := filepath.Clean(m.root + "/" + path)
	if _, ok := m.content[resolved]; ok {
		return resolved, nil
	}
	// Try as absolute path with leading slash.
	absolute := "/" + path
	if _, ok := m.content[absolute]; ok {
		return absolute, nil
	}
	return "", fmt.Errorf("unknown resolved path: '%s'", resolved)
}

func (m *mutableLoader) Inherit(from string) (loaders.Loader, error) {
	root := m.root
	if from != "" {
		resolvedFrom, err := m.Resolve(from)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve '%s': %w", from, err)
		}
		components := strings.Split(resolvedFrom, "/")
		if len(components) < 2 {
			root = "/"
		} else {
			root = strings.Join(components[:len(components)-1], "/")
		}
	}
	return &mutableLoader{
		content: m.content,
		root:    root,
	}, nil
}
