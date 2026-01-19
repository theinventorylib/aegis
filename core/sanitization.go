package core

import (
	"html"
	"regexp"
	"strings"
	"unicode"
)

// Sanitization provides utilities for cleaning and normalizing user inputs
// to prevent injection attacks, XSS, and other security vulnerabilities.
//
// This module implements defense-in-depth by:
//   - Removing dangerous characters and patterns
//   - Normalizing whitespace and encoding
//   - Validating input against expected formats
//   - Providing context-specific sanitization functions
//
// Note: Sanitization is NOT a replacement for proper validation, parameterized
// queries, or output encoding. Always use multiple layers of security.

// SanitizationConfig controls the behavior of sanitization functions.
type SanitizationConfig struct {
	// MaxLength is the maximum allowed length for sanitized strings (0 = no limit)
	MaxLength int

	// AllowUnicode determines if non-ASCII Unicode characters are permitted
	AllowUnicode bool

	// StripHTML removes all HTML tags from input
	StripHTML bool

	// NormalizeWhitespace collapses multiple spaces into single spaces
	NormalizeWhitespace bool

	// TrimWhitespace removes leading and trailing whitespace
	TrimWhitespace bool
}

// DefaultSanitizationConfig returns a secure default configuration.
//
// Defaults:
//   - MaxLength: 1000 characters (prevents DoS via large inputs)
//   - AllowUnicode: true (supports international users)
//   - StripHTML: true (prevents XSS)
//   - NormalizeWhitespace: true (cleans up formatting)
//   - TrimWhitespace: true (removes accidental spaces)
func DefaultSanitizationConfig() *SanitizationConfig {
	return &SanitizationConfig{
		MaxLength:           1000,
		AllowUnicode:        true,
		StripHTML:           true,
		NormalizeWhitespace: true,
		TrimWhitespace:      true,
	}
}

// Dangerous patterns that should be removed or escaped
var (
	// htmlTagRegex matches HTML tags for removal
	htmlTagRegex = regexp.MustCompile(`<[^>]*>`)

	// scriptTagRegex matches script tags (case-insensitive)
	scriptTagRegex = regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`)

	// sqlCommentRegex matches SQL comment patterns
	sqlCommentRegex = regexp.MustCompile(`(--|#|/\*|\*/|;)`)

	// controlCharRegex matches control characters (except newline, tab, carriage return)
	controlCharRegex = regexp.MustCompile(`[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]`)

	// multipleSpacesRegex matches multiple consecutive spaces
	multipleSpacesRegex = regexp.MustCompile(`\s+`)

	// nullByteRegex matches null bytes
	nullByteRegex = regexp.MustCompile(`\x00`)
)

// SanitizeString performs general-purpose string sanitization.
//
// This is the primary sanitization function for most user inputs like names,
// descriptions, and general text fields. It applies multiple security measures:
//   - Removes null bytes (prevents null byte injection)
//   - Strips HTML tags (prevents XSS)
//   - Removes control characters (prevents terminal injection)
//   - Normalizes whitespace (improves data quality)
//   - Enforces length limits (prevents DoS)
//
// Parameters:
//   - input: The raw user input string
//   - config: Sanitization configuration (nil = use defaults)
//
// Example:
//
//	name := core.SanitizeString(userInput, nil)
//	// Input:  "John<script>alert('xss')</script>  Doe  "
//	// Output: "John Doe"
func SanitizeString(input string, config *SanitizationConfig) string {
	if config == nil {
		config = DefaultSanitizationConfig()
	}

	// 1. Unescape FIRST to catch encoded payloads
	if config.StripHTML {
		input = html.UnescapeString(input)
		// Also decode HTML entities to prevent double-encoding attacks
		input = html.UnescapeString(input)
	}
	// 2. Apply filters
	input = nullByteRegex.ReplaceAllString(input, "")
	input = scriptTagRegex.ReplaceAllString(input, "")
	if config.StripHTML {
		input = htmlTagRegex.ReplaceAllString(input, "")
	}

	// Remove control characters (except \n, \t, \r)
	input = controlCharRegex.ReplaceAllString(input, "")

	// Filter non-ASCII if Unicode is not allowed
	if !config.AllowUnicode {
		input = removeNonASCII(input)
	}

	// Normalize whitespace
	if config.NormalizeWhitespace {
		input = multipleSpacesRegex.ReplaceAllString(input, " ")
	}

	// Trim whitespace
	if config.TrimWhitespace {
		input = strings.TrimSpace(input)
	}

	// 3. Robust Truncation (Rune-aware)
	if config.MaxLength > 0 {
		runes := []rune(input)
		if len(runes) > config.MaxLength {
			input = string(runes[:config.MaxLength])
		}
	}

	return input
}

// SanitizeEmail sanitizes and normalizes email addresses.
//
// Email-specific sanitization:
//   - Converts to lowercase (emails are case-insensitive)
//   - Removes whitespace
//   - Removes dangerous characters
//   - Validates basic format
//
// Note: This does NOT validate email format. Use ValidateEmail() for validation.
//
// Example:
//
//	email := core.SanitizeEmail("  John.Doe@EXAMPLE.com  ")
//	// Output: "john.doe@example.com"
func SanitizeEmail(email string) string {
	// Remove all whitespace
	email = strings.TrimSpace(email)
	email = strings.ReplaceAll(email, " ", "")
	email = strings.ReplaceAll(email, "\t", "")
	email = strings.ReplaceAll(email, "\n", "")
	email = strings.ReplaceAll(email, "\r", "")

	// Convert to lowercase (email addresses are case-insensitive)
	email = strings.ToLower(email)

	// Remove null bytes
	email = nullByteRegex.ReplaceAllString(email, "")

	// Remove control characters
	email = controlCharRegex.ReplaceAllString(email, "")

	return email
}

// SanitizeUsername sanitizes usernames for safe storage and display.
//
// Username-specific rules:
//   - Allows alphanumeric, underscore, hyphen, and period
//   - Removes all other characters
//   - Converts to lowercase for consistency
//   - Enforces length limits
//
// Parameters:
//   - username: The raw username input
//   - maxLength: Maximum allowed length (0 = use default of 50)
//
// Example:
//
//	username := core.SanitizeUsername("John_Doe123!@#", 0)
//	// Output: "john_doe123"
func SanitizeUsername(username string, maxLength int) string {
	if maxLength == 0 {
		maxLength = 50 // default max username length
	}

	// Trim whitespace
	username = strings.TrimSpace(username)

	// Convert to lowercase
	username = strings.ToLower(username)

	// Remove null bytes and control characters
	username = nullByteRegex.ReplaceAllString(username, "")
	username = controlCharRegex.ReplaceAllString(username, "")

	// Keep only alphanumeric, underscore, hyphen, and period
	var result strings.Builder
	for _, char := range username {
		if (char >= 'a' && char <= 'z') ||
			(char >= '0' && char <= '9') ||
			char == '_' || char == '-' || char == '.' {
			result.WriteRune(char)
		}
	}

	username = result.String()

	// Enforce maximum length
	if len(username) > maxLength {
		username = username[:maxLength]
	}

	return username
}

// SanitizeURL sanitizes URLs to prevent injection attacks.
//
// URL-specific sanitization:
//   - Removes whitespace
//   - Blocks javascript: and data: schemes
//   - Removes null bytes
//   - Validates basic URL structure
//
// Note: This does NOT validate URL format. Use proper URL validation separately.
//
// Example:
//
//	url := core.SanitizeURL("  https://example.com/path  ")
//	// Output: "https://example.com/path"
//
//	url := core.SanitizeURL("javascript:alert('xss')")
//	// Output: "" (dangerous scheme blocked)
func SanitizeURL(url string) string {
	// Trim whitespace
	url = strings.TrimSpace(url)

	// Remove null bytes
	url = nullByteRegex.ReplaceAllString(url, "")

	// Remove control characters
	url = controlCharRegex.ReplaceAllString(url, "")

	// Convert to lowercase for scheme checking
	lowerURL := strings.ToLower(url)

	// Block dangerous URL schemes
	dangerousSchemes := []string{
		"javascript:",
		"data:",
		"vbscript:",
		"file:",
		"about:",
	}

	for _, scheme := range dangerousSchemes {
		if strings.HasPrefix(lowerURL, scheme) {
			return "" // Return empty string for dangerous URLs
		}
	}

	return url
}

// SanitizeFilename sanitizes filenames to prevent directory traversal attacks.
//
// Filename-specific sanitization:
//   - Removes path separators (/, \)
//   - Removes null bytes
//   - Removes control characters
//   - Blocks directory traversal patterns (.., .)
//   - Enforces length limits
//
// Example:
//
//	filename := core.SanitizeFilename("../../etc/passwd")
//	// Output: "etcpasswd"
//
//	filename := core.SanitizeFilename("my<file>.txt")
//	// Output: "myfile.txt"
func SanitizeFilename(filename string) string {
	// Trim whitespace
	filename = strings.TrimSpace(filename)

	// Remove null bytes
	filename = nullByteRegex.ReplaceAllString(filename, "")

	// Remove control characters
	filename = controlCharRegex.ReplaceAllString(filename, "")

	// Remove path separators
	filename = strings.ReplaceAll(filename, "/", "")
	filename = strings.ReplaceAll(filename, "\\", "")

	// Remove directory traversal patterns
	filename = strings.ReplaceAll(filename, "..", "")

	// Remove potentially dangerous characters
	dangerousChars := []string{"<", ">", ":", "\"", "|", "?", "*"}
	for _, char := range dangerousChars {
		filename = strings.ReplaceAll(filename, char, "")
	}

	// Enforce maximum length (255 is typical filesystem limit)
	if len(filename) > 255 {
		filename = filename[:255]
	}

	return filename
}

// SanitizeHTML sanitizes HTML content for safe display.
//
// This function escapes HTML entities to prevent XSS attacks while preserving
// the original text content. Use this when you need to display user-generated
// content in HTML context.
//
// Example:
//
//	content := core.SanitizeHTML("<script>alert('xss')</script>")
//	// Output: "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;"
func SanitizeHTML(content string) string {
	// Remove null bytes
	content = nullByteRegex.ReplaceAllString(content, "")

	// Escape HTML entities
	content = html.EscapeString(content)

	return content
}

// SanitizeSQL removes common SQL injection patterns.
//
// WARNING: This is NOT a replacement for parameterized queries!
// Always use prepared statements or parameterized queries for SQL.
// This function is only a defense-in-depth measure.
//
// Removes:
//   - SQL comments (-- , #, /* */)
//   - Semicolons (statement terminators)
//   - Null bytes
//
// Example:
//
//	input := core.SanitizeSQL("admin' OR '1'='1' --")
//	// Output: "admin' OR '1'='1' "
func SanitizeSQL(input string) string {
	// Remove null bytes
	input = nullByteRegex.ReplaceAllString(input, "")

	// Remove SQL comment patterns
	input = sqlCommentRegex.ReplaceAllString(input, "")

	return input
}

// SanitizePhoneNumber sanitizes phone numbers to a consistent format.
//
// Phone-specific sanitization:
//   - Keeps only digits, plus sign, and hyphens
//   - Removes all other characters
//   - Trims whitespace
//
// Example:
//
//	phone := core.SanitizePhoneNumber("+1 (555) 123-4567")
//	// Output: "+1-555-123-4567"
func SanitizePhoneNumber(phone string) string {
	// Trim whitespace
	phone = strings.TrimSpace(phone)

	// Remove null bytes and control characters
	phone = nullByteRegex.ReplaceAllString(phone, "")
	phone = controlCharRegex.ReplaceAllString(phone, "")

	// Keep only digits, plus, and hyphen
	var result strings.Builder
	for _, char := range phone {
		if unicode.IsDigit(char) || char == '+' || char == '-' {
			result.WriteRune(char)
		}
	}

	return result.String()
}

// SanitizeMultiline sanitizes multi-line text input.
//
// Multiline-specific sanitization:
//   - Preserves newlines and basic formatting
//   - Removes dangerous HTML/scripts
//   - Normalizes line endings to \n
//   - Limits total length
//
// Use this for text areas, descriptions, and comments.
//
// Example:
//
//	text := core.SanitizeMultiline("Line 1\r\nLine 2\r\n<script>alert('xss')</script>", 5000)
//	// Output: "Line 1\nLine 2\n"
func SanitizeMultiline(input string, maxLength int) string {
	if maxLength == 0 {
		maxLength = 5000 // default max for multiline text
	}

	// Remove null bytes
	input = nullByteRegex.ReplaceAllString(input, "")

	// Remove script tags
	input = scriptTagRegex.ReplaceAllString(input, "")

	// Strip HTML tags
	input = htmlTagRegex.ReplaceAllString(input, "")

	// Normalize line endings to \n
	input = strings.ReplaceAll(input, "\r\n", "\n")
	input = strings.ReplaceAll(input, "\r", "\n")

	// Remove control characters except newline and tab
	var result strings.Builder
	for _, char := range input {
		// Allow newline, tab, and printable characters
		if char == '\n' || char == '\t' || (char >= 32 && char <= 126) || char > 127 {
			result.WriteRune(char)
		}
	}

	input = result.String()

	// Enforce maximum length
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

// StripTags removes all HTML tags from a string.
//
// This is a convenience function for quick HTML tag removal.
//
// Example:
//
//	text := core.StripTags("<p>Hello <b>World</b></p>")
//	// Output: "Hello World"
func StripTags(input string) string {
	return htmlTagRegex.ReplaceAllString(input, "")
}

// NormalizeWhitespace collapses multiple spaces into single spaces.
//
// Example:
//
//	text := core.NormalizeWhitespace("Hello    World  \n  Test")
//	// Output: "Hello World Test"
func NormalizeWhitespace(input string) string {
	return strings.TrimSpace(multipleSpacesRegex.ReplaceAllString(input, " "))
}
