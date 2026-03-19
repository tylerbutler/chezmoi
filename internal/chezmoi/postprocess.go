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
