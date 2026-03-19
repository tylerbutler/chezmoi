package chezmoi

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"text/template"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/unicode"
)

// TemplateOptions are template options that can be set with directives.
type TemplateOptions struct {
	Encoding       encoding.Encoding
	Funcs          template.FuncMap
	FormatIndent   string
	LeftDelimiter  string
	LineEnding     string
	RightDelimiter string
	Options        []string
}

// ParseTemplate parses a template named name from data with the given options
// and returns a GoTemplate.
func ParseTemplate(name string, data []byte, options TemplateOptions) (*GoTemplate, error) {
	t := &GoTemplate{}
	if err := t.Parse(name, data, options); err != nil {
		return nil, err
	}
	return t, nil
}

// parseAndRemoveDirectives updates o by parsing all template directives in data
// and returns data with the lines containing directives removed. The lines are
// removed so that any delimiters do not break template parsing.
func (o *TemplateOptions) parseAndRemoveDirectives(data []byte) ([]byte, error) {
	directiveMatches := templateDirectiveRx.FindAllSubmatchIndex(data, -1)
	if directiveMatches == nil {
		return data, nil
	}

	// Parse options from directives.
	for _, directiveMatch := range directiveMatches {
		keyValuePairMatches := templateDirectiveKeyValuePairRx.FindAllSubmatch(data[directiveMatch[2]:directiveMatch[3]], -1)
		for _, keyValuePairMatch := range keyValuePairMatches {
			key := string(keyValuePairMatch[1])
			value := maybeUnquote(string(keyValuePairMatch[2]))
			switch key {
			case "encoding":
				switch value {
				case "utf-8":
					o.Encoding = unicode.UTF8
				case "utf-8-bom":
					o.Encoding = unicode.UTF8BOM
				case "utf-16-be":
					o.Encoding = unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM)
				case "utf-16-be-bom":
					o.Encoding = unicode.UTF16(unicode.BigEndian, unicode.UseBOM)
				case "utf-16-le":
					o.Encoding = unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM)
				case "utf-16-le-bom":
					o.Encoding = unicode.UTF16(unicode.LittleEndian, unicode.UseBOM)
				default:
					return nil, fmt.Errorf("%s: unknown encoding", value)
				}
			case "format-indent":
				o.FormatIndent = value
			case "format-indent-width":
				width, err := strconv.Atoi(value)
				if err != nil {
					return nil, err
				}
				o.FormatIndent = strings.Repeat(" ", width)
			case "left-delimiter":
				o.LeftDelimiter = value
			case "line-ending", "line-endings":
				switch string(keyValuePairMatch[2]) {
				case "crlf":
					o.LineEnding = "\r\n"
				case "lf":
					o.LineEnding = "\n"
				case "native":
					o.LineEnding = nativeLineEnding
				default:
					o.LineEnding = value
				}
			case "right-delimiter":
				o.RightDelimiter = value
			case "missing-key":
				o.Options = append(o.Options, "missingkey="+value)
			}
		}
	}

	return removeMatches(data, directiveMatches), nil
}

// removeMatches returns data with matchesIndexes removed.
func removeMatches(data []byte, matchesIndexes [][]int) []byte {
	slices := make([][]byte, len(matchesIndexes)+1)
	slices[0] = data[:matchesIndexes[0][0]]
	for i, matchIndexes := range matchesIndexes[1:] {
		slices[i+1] = data[matchesIndexes[i][1]:matchIndexes[0]]
	}
	slices[len(matchesIndexes)] = data[matchesIndexes[len(matchesIndexes)-1][1]:]
	return bytes.Join(slices, nil)
}

// replaceLineEndings replaces all line endings in s with lineEnding. If
// lineEnding is empty it returns s unchanged.
func replaceLineEndings(s, lineEnding string) string {
	if lineEnding == "" {
		return s
	}
	return lineEndingRx.ReplaceAllString(s, lineEnding)
}
