package core

import (
	"strings"
	"testing"
)

func TestSanitizeString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		config   *SanitizationConfig
		expected string
	}{
		{
			name:     "removes HTML tags",
			input:    "Hello <script>alert('xss')</script> World",
			config:   nil,
			expected: "Hello World",
		},
		{
			name:     "removes null bytes",
			input:    "Hello\x00World",
			config:   nil,
			expected: "HelloWorld",
		},
		{
			name:     "normalizes whitespace",
			input:    "Hello    World  \n  Test",
			config:   nil,
			expected: "Hello World Test",
		},
		{
			name:     "trims whitespace",
			input:    "  Hello World  ",
			config:   nil,
			expected: "Hello World",
		},
		{
			name:  "enforces max length",
			input: "This is a very long string that should be truncated",
			config: &SanitizationConfig{
				MaxLength:      10,
				TrimWhitespace: true,
			},
			expected: "This is a ",
		},
		{
			name:  "removes non-ASCII when configured",
			input: "Hello 世界",
			config: &SanitizationConfig{
				AllowUnicode:   false,
				TrimWhitespace: true,
			},
			expected: "Hello",
		},
		{
			name:     "preserves Unicode by default",
			input:    "Hello 世界",
			config:   nil,
			expected: "Hello 世界",
		},
		{
			name:     "removes control characters",
			input:    "Hello\x01\x02\x03World",
			config:   nil,
			expected: "HelloWorld",
		},
		{
			name:     "handles empty string",
			input:    "",
			config:   nil,
			expected: "",
		},
		{
			name:     "handles only whitespace",
			input:    "   \n\t  ",
			config:   nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeString(tt.input, tt.config)
			if result != tt.expected {
				t.Errorf("SanitizeString() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestSanitizeEmail(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "converts to lowercase",
			input:    "John.Doe@EXAMPLE.COM",
			expected: "john.doe@example.com",
		},
		{
			name:     "removes whitespace",
			input:    "  john@example.com  ",
			expected: "john@example.com",
		},
		{
			name:     "removes internal whitespace",
			input:    "john @example.com",
			expected: "john@example.com",
		},
		{
			name:     "removes null bytes",
			input:    "john\x00@example.com",
			expected: "john@example.com",
		},
		{
			name:     "removes control characters",
			input:    "john\x01@example.com",
			expected: "john@example.com",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "removes newlines and tabs",
			input:    "john\n@\texample.com",
			expected: "john@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeEmail(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeEmail() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestSanitizeUsername(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		maxLength int
		expected  string
	}{
		{
			name:      "allows alphanumeric and underscore",
			input:     "john_doe123",
			maxLength: 0,
			expected:  "john_doe123",
		},
		{
			name:      "removes special characters",
			input:     "john!@#$%doe",
			maxLength: 0,
			expected:  "johndoe",
		},
		{
			name:      "converts to lowercase",
			input:     "JohnDoe",
			maxLength: 0,
			expected:  "johndoe",
		},
		{
			name:      "allows hyphens and periods",
			input:     "john-doe.123",
			maxLength: 0,
			expected:  "john-doe.123",
		},
		{
			name:      "enforces max length",
			input:     "verylongusername",
			maxLength: 8,
			expected:  "verylong",
		},
		{
			name:      "removes null bytes",
			input:     "john\x00doe",
			maxLength: 0,
			expected:  "johndoe",
		},
		{
			name:      "handles empty string",
			input:     "",
			maxLength: 0,
			expected:  "",
		},
		{
			name:      "removes whitespace",
			input:     "  john doe  ",
			maxLength: 0,
			expected:  "johndoe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeUsername(tt.input, tt.maxLength)
			if result != tt.expected {
				t.Errorf("SanitizeUsername() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestSanitizeURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "allows valid HTTP URL",
			input:    "https://example.com/path",
			expected: "https://example.com/path",
		},
		{
			name:     "blocks javascript scheme",
			input:    "javascript:alert('xss')",
			expected: "",
		},
		{
			name:     "blocks data scheme",
			input:    "data:text/html,<script>alert('xss')</script>",
			expected: "",
		},
		{
			name:     "blocks vbscript scheme",
			input:    "vbscript:msgbox('xss')",
			expected: "",
		},
		{
			name:     "blocks file scheme",
			input:    "file:///etc/passwd",
			expected: "",
		},
		{
			name:     "removes whitespace",
			input:    "  https://example.com  ",
			expected: "https://example.com",
		},
		{
			name:     "removes null bytes",
			input:    "https://example.com\x00/path",
			expected: "https://example.com/path",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "case insensitive scheme blocking",
			input:    "JaVaScRiPt:alert('xss')",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeURL(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeURL() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "allows simple filename",
			input:    "document.pdf",
			expected: "document.pdf",
		},
		{
			name:     "removes path separators",
			input:    "../../etc/passwd",
			expected: "etcpasswd",
		},
		{
			name:     "removes directory traversal",
			input:    "..\\..\\windows\\system32",
			expected: "windowssystem32",
		},
		{
			name:     "removes dangerous characters",
			input:    "file<name>.txt",
			expected: "filename.txt",
		},
		{
			name:     "removes null bytes",
			input:    "file\x00name.txt",
			expected: "filename.txt",
		},
		{
			name:     "enforces max length",
			input:    strings.Repeat("a", 300) + ".txt",
			expected: strings.Repeat("a", 255),
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "removes pipes and wildcards",
			input:    "file|name*.txt",
			expected: "filename.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeFilename() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestSanitizeHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "escapes HTML tags",
			input:    "<script>alert('xss')</script>",
			expected: "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;",
		},
		{
			name:     "escapes special characters",
			input:    "Hello & goodbye",
			expected: "Hello &amp; goodbye",
		},
		{
			name:     "escapes quotes",
			input:    `"Hello" 'World'`,
			expected: "&#34;Hello&#34; &#39;World&#39;",
		},
		{
			name:     "removes null bytes",
			input:    "Hello\x00World",
			expected: "HelloWorld",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeHTML(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeHTML() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestSanitizeSQL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes SQL comments (--)",
			input:    "admin' OR '1'='1' --",
			expected: "admin' OR '1'='1' ",
		},
		{
			name:     "removes SQL comments (#)",
			input:    "admin' # comment",
			expected: "admin'  comment",
		},
		{
			name:     "removes block comments",
			input:    "admin' /* comment */ OR '1'='1'",
			expected: "admin'  comment  OR '1'='1'",
		},
		{
			name:     "removes semicolons",
			input:    "admin'; DROP TABLE users;",
			expected: "admin' DROP TABLE users",
		},
		{
			name:     "removes null bytes",
			input:    "admin\x00' OR '1'='1'",
			expected: "admin' OR '1'='1'",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeSQL(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeSQL() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestSanitizePhoneNumber(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "keeps digits and formatting",
			input:    "+1-555-123-4567",
			expected: "+1-555-123-4567",
		},
		{
			name:     "removes parentheses and spaces",
			input:    "+1 (555) 123-4567",
			expected: "+1555123-4567",
		},
		{
			name:     "removes letters",
			input:    "1-800-FLOWERS",
			expected: "1-800-",
		},
		{
			name:     "removes null bytes",
			input:    "+1\x00-555-1234",
			expected: "+1-555-1234",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "removes special characters",
			input:    "+1.555.123.4567",
			expected: "+15551234567",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizePhoneNumber(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizePhoneNumber() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestSanitizeMultiline(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		maxLength int
		expected  string
	}{
		{
			name:      "preserves newlines",
			input:     "Line 1\nLine 2\nLine 3",
			maxLength: 0,
			expected:  "Line 1\nLine 2\nLine 3",
		},
		{
			name:      "normalizes line endings",
			input:     "Line 1\r\nLine 2\rLine 3",
			maxLength: 0,
			expected:  "Line 1\nLine 2\nLine 3",
		},
		{
			name:      "removes script tags",
			input:     "Hello\n<script>alert('xss')</script>\nWorld",
			maxLength: 0,
			expected:  "Hello\n\nWorld",
		},
		{
			name:      "removes HTML tags",
			input:     "Hello <b>World</b>\n<p>Test</p>",
			maxLength: 0,
			expected:  "Hello World\nTest",
		},
		{
			name:      "enforces max length",
			input:     "This is a very long multiline text",
			maxLength: 10,
			expected:  "This is a ",
		},
		{
			name:      "removes null bytes",
			input:     "Hello\x00\nWorld",
			maxLength: 0,
			expected:  "Hello\nWorld",
		},
		{
			name:      "handles empty string",
			input:     "",
			maxLength: 0,
			expected:  "",
		},
		{
			name:      "preserves tabs",
			input:     "Line 1\n\tIndented line",
			maxLength: 0,
			expected:  "Line 1\n\tIndented line",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeMultiline(tt.input, tt.maxLength)
			if result != tt.expected {
				t.Errorf("SanitizeMultiline() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestStripTags(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes simple tags",
			input:    "<p>Hello World</p>",
			expected: "Hello World",
		},
		{
			name:     "removes nested tags",
			input:    "<div><p>Hello <b>World</b></p></div>",
			expected: "Hello World",
		},
		{
			name:     "removes self-closing tags",
			input:    "Hello<br/>World",
			expected: "HelloWorld",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StripTags(tt.input)
			if result != tt.expected {
				t.Errorf("StripTags() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestNormalizeWhitespace(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "collapses multiple spaces",
			input:    "Hello    World",
			expected: "Hello World",
		},
		{
			name:     "removes leading and trailing whitespace",
			input:    "  Hello World  ",
			expected: "Hello World",
		},
		{
			name:     "handles tabs and newlines",
			input:    "Hello\t\n\nWorld",
			expected: "Hello World",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "handles only whitespace",
			input:    "   \n\t  ",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeWhitespace(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeWhitespace() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestDefaultSanitizationConfig(t *testing.T) {
	config := DefaultSanitizationConfig()

	if config.MaxLength != 1000 {
		t.Errorf("DefaultSanitizationConfig().MaxLength = %d, want 1000", config.MaxLength)
	}
	if !config.AllowUnicode {
		t.Error("DefaultSanitizationConfig().AllowUnicode = false, want true")
	}
	if !config.StripHTML {
		t.Error("DefaultSanitizationConfig().StripHTML = false, want true")
	}
	if !config.NormalizeWhitespace {
		t.Error("DefaultSanitizationConfig().NormalizeWhitespace = false, want true")
	}
	if !config.TrimWhitespace {
		t.Error("DefaultSanitizationConfig().TrimWhitespace = false, want true")
	}
}

// Benchmark tests
func BenchmarkSanitizeString(b *testing.B) {
	input := "Hello <script>alert('xss')</script> World   with   spaces"
	for i := 0; i < b.N; i++ {
		SanitizeString(input, nil)
	}
}

func BenchmarkSanitizeEmail(b *testing.B) {
	input := "  John.Doe@EXAMPLE.COM  "
	for i := 0; i < b.N; i++ {
		SanitizeEmail(input)
	}
}

func BenchmarkSanitizeUsername(b *testing.B) {
	input := "John_Doe123!@#$%"
	for i := 0; i < b.N; i++ {
		SanitizeUsername(input, 0)
	}
}

func BenchmarkSanitizeHTML(b *testing.B) {
	input := "<div><p>Hello <b>World</b></p></div>"
	for i := 0; i < b.N; i++ {
		SanitizeHTML(input)
	}
}
