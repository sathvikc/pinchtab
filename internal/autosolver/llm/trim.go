package llm

import (
	"regexp"
	"strings"
)

// TrimHTML strips non-essential content from HTML to reduce token usage.
// It removes script tags, style tags, comments, excessive whitespace,
// and non-interactive elements that don't contribute to page understanding.
//
// The goal is to reduce a full page (~50-200KB) to ~4KB while preserving
// interactive elements (forms, buttons, inputs, links) and their attributes.
func TrimHTML(html string) string {
	// Remove script tags and their content.
	html = reScript.ReplaceAllString(html, "")

	// Remove style tags and their content.
	html = reStyle.ReplaceAllString(html, "")

	// Remove HTML comments.
	html = reComment.ReplaceAllString(html, "")

	// Remove SVG elements (often large and not useful for automation).
	html = reSVG.ReplaceAllString(html, "")

	// Remove data URIs (base64 images inflate size dramatically).
	html = reDataURI.ReplaceAllString(html, `""`)

	// Collapse excessive whitespace.
	html = reWhitespace.ReplaceAllString(html, " ")

	// Collapse multiple newlines.
	html = reNewlines.ReplaceAllString(html, "\n")

	// Trim leading/trailing whitespace on each line.
	lines := strings.Split(html, "\n")
	var trimmed []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			trimmed = append(trimmed, line)
		}
	}
	html = strings.Join(trimmed, "\n")

	// Hard cap to prevent excessive token usage.
	if len(html) > maxTrimmedSize {
		html = html[:maxTrimmedSize]
	}

	return html
}

const maxTrimmedSize = 4000

var (
	reScript     = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	reStyle      = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	reComment    = regexp.MustCompile(`(?s)<!--.*?-->`)
	reSVG        = regexp.MustCompile(`(?is)<svg[^>]*>.*?</svg>`)
	reDataURI    = regexp.MustCompile(`"data:[^"]*"`)
	reWhitespace = regexp.MustCompile(`[ \t]+`)
	reNewlines   = regexp.MustCompile(`\n{3,}`)
)
