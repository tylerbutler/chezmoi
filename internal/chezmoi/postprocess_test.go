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
