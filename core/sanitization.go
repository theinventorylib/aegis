package core

import (
	"html"
	"regexp"
	"strings"
	"unicode"
)

// SanitizationConfig controls the behavior of sanitization functions.
// Sanitization is NOT a replacement for validation, parameterized queries,
// or output encoding — it is one layer of defense.
type SanitizationConfig struct {
	MaxLength int // maximum length (0 = no limit)

	AllowUnicode bool // permits non-ASCII Unicode characters
	StripHTML    bool // removes all HTML tags from input

	// NormalizeWhitespace collapses runs of whitespace into single spaces.
	NormalizeWhitespace bool
	TrimWhitespace      bool // removes leading/trailing whitespace
}

// DefaultSanitizationConfig returns a secure default configuration
// (1000-char limit, Unicode allowed, HTML stripped, whitespace normalized).
func DefaultSanitizationConfig() *SanitizationConfig {
	return &SanitizationConfig{
		MaxLength:           1000,
		AllowUnicode:        true,
		StripHTML:           true,
		NormalizeWhitespace: true,
		TrimWhitespace:      true,
	}
}

var (
	htmlTagRegex        = regexp.MustCompile(`<[^>]*>`)
	scriptTagRegex      = regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`)
	sqlCommentRegex     = regexp.MustCompile(`(--|#|/\*|\*/|;)`)
	controlCharRegex    = regexp.MustCompile(`[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]`)
	multipleSpacesRegex = regexp.MustCompile(`\s+`)
	nullByteRegex       = regexp.MustCompile(`\x00`)
)

// SanitizeString is the primary sanitization function for user inputs like
// names and descriptions: it removes null bytes and control characters,
// optionally unescapes/strips HTML, normalizes whitespace, and enforces
// MaxLength (rune-aware). config nil = DefaultSanitizationConfig.
func SanitizeString(input string, config *SanitizationConfig) string {
	if config == nil {
		config = DefaultSanitizationConfig()
	}

	// Unescape first to catch encoded payloads; twice to prevent
	// double-encoding attacks.
	if config.StripHTML {
		input = html.UnescapeString(input)
		input = html.UnescapeString(input)
	}
	input = nullByteRegex.ReplaceAllString(input, "")
	input = scriptTagRegex.ReplaceAllString(input, "")
	if config.StripHTML {
		input = htmlTagRegex.ReplaceAllString(input, "")
	}

	input = controlCharRegex.ReplaceAllString(input, "")

	if !config.AllowUnicode {
		input = removeNonASCII(input)
	}
	if config.NormalizeWhitespace {
		input = multipleSpacesRegex.ReplaceAllString(input, " ")
	}
	if config.TrimWhitespace {
		input = strings.TrimSpace(input)
	}

	// Rune-aware truncation.
	if config.MaxLength > 0 {
		runes := []rune(input)
		if len(runes) > config.MaxLength {
			input = string(runes[:config.MaxLength])
		}
	}

	return input
}

// SanitizeEmail normalizes an email address: strips whitespace, lowercases
// (emails are case-insensitive), and removes null/control bytes. It does NOT
// validate format — use ValidateEmail for that.
func SanitizeEmail(email string) string {
	email = strings.TrimSpace(email)
	email = strings.ReplaceAll(email, " ", "")
	email = strings.ReplaceAll(email, "\t", "")
	email = strings.ReplaceAll(email, "\n", "")
	email = strings.ReplaceAll(email, "\r", "")
	email = strings.ToLower(email)
	email = nullByteRegex.ReplaceAllString(email, "")
	email = controlCharRegex.ReplaceAllString(email, "")
	return email
}

// SanitizeUsername sanitizes usernames: keeps only [a-z0-9_.-], lowercases,
// and trims to maxLength (0 = default of 50).
func SanitizeUsername(username string, maxLength int) string {
	if maxLength == 0 {
		maxLength = 50
	}

	username = strings.TrimSpace(username)
	username = strings.ToLower(username)
	username = nullByteRegex.ReplaceAllString(username, "")
	username = controlCharRegex.ReplaceAllString(username, "")

	var result strings.Builder
	for _, char := range username {
		if (char >= 'a' && char <= 'z') ||
			(char >= '0' && char <= '9') ||
			char == '_' || char == '-' || char == '.' {
			result.WriteRune(char)
		}
	}
	username = result.String()

	if len(username) > maxLength {
		username = username[:maxLength]
	}
	return username
}

// SanitizeURL trims whitespace, removes null/control bytes, and returns ""
// for dangerous schemes (javascript:, data:, vbscript:, file:, about:).
func SanitizeURL(url string) string {
	url = strings.TrimSpace(url)
	url = nullByteRegex.ReplaceAllString(url, "")
	url = controlCharRegex.ReplaceAllString(url, "")

	lowerURL := strings.ToLower(url)
	dangerousSchemes := []string{"javascript:", "data:", "vbscript:", "file:", "about:"}
	for _, scheme := range dangerousSchemes {
		if strings.HasPrefix(lowerURL, scheme) {
			return ""
		}
	}
	return url
}

// sanitizeFilename sanitizes filenames to prevent directory traversal:
// strips path separators, traversal patterns, and dangerous characters,
// capping at 255 bytes.
func sanitizeFilename(filename string) string {
	filename = strings.TrimSpace(filename)
	filename = nullByteRegex.ReplaceAllString(filename, "")
	filename = controlCharRegex.ReplaceAllString(filename, "")
	filename = strings.ReplaceAll(filename, "/", "")
	filename = strings.ReplaceAll(filename, "\\", "")
	filename = strings.ReplaceAll(filename, "..", "")

	for _, char := range []string{"<", ">", ":", "\"", "|", "?", "*"} {
		filename = strings.ReplaceAll(filename, char, "")
	}
	if len(filename) > 255 {
		filename = filename[:255]
	}
	return filename
}

// sanitizeHTML escapes HTML entities (and removes null bytes) so
// user-generated content can be displayed safely in HTML context.
func sanitizeHTML(content string) string {
	content = nullByteRegex.ReplaceAllString(content, "")
	return html.EscapeString(content)
}

// sanitizeSQL strips SQL comment patterns and null bytes as a
// defense-in-depth measure. NOT a replacement for parameterized queries.
func sanitizeSQL(input string) string {
	input = nullByteRegex.ReplaceAllString(input, "")
	return sqlCommentRegex.ReplaceAllString(input, "")
}

// SanitizePhoneNumber sanitizes phone numbers to digits, plus sign, and
// hyphens only:
//
//	core.SanitizePhoneNumber("+1 (555) 123-4567") // "+1555123-4567"
func SanitizePhoneNumber(phone string) string {
	phone = strings.TrimSpace(phone)
	phone = nullByteRegex.ReplaceAllString(phone, "")
	phone = controlCharRegex.ReplaceAllString(phone, "")

	var result strings.Builder
	for _, char := range phone {
		if unicode.IsDigit(char) || char == '+' || char == '-' {
			result.WriteRune(char)
		}
	}
	return result.String()
}

// SanitizeMultiline sanitizes multi-line text (text areas, descriptions):
// strips scripts/HTML, normalizes line endings to \n, keeps newlines/tabs,
// and caps at maxLength (0 = default 5000).
func SanitizeMultiline(input string, maxLength int) string {
	if maxLength == 0 {
		maxLength = 5000
	}

	input = nullByteRegex.ReplaceAllString(input, "")
	input = scriptTagRegex.ReplaceAllString(input, "")
	input = htmlTagRegex.ReplaceAllString(input, "")

	input = strings.ReplaceAll(input, "\r\n", "\n")
	input = strings.ReplaceAll(input, "\r", "\n")

	// Keep newline, tab, and printable characters.
	var result strings.Builder
	for _, char := range input {
		if char == '\n' || char == '\t' || (char >= 32 && char <= 126) || char > 127 {
			result.WriteRune(char)
		}
	}
	input = result.String()

	if len(input) > maxLength {
		input = input[:maxLength]
	}
	return input
}

// removeNonASCII removes all non-ASCII characters from a string.
func removeNonASCII(input string) string {
	var result strings.Builder
	for _, char := range input {
		if char <= 127 {
			result.WriteRune(char)
		}
	}
	return result.String()
}

// stripTags removes all HTML tags from a string.
func stripTags(input string) string {
	return htmlTagRegex.ReplaceAllString(input, "")
}

// normalizeWhitespace collapses runs of whitespace into single spaces.
func normalizeWhitespace(input string) string {
	return strings.TrimSpace(multipleSpacesRegex.ReplaceAllString(input, " "))
}
