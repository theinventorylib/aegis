// This example demonstrates how to use the sanitization module in Aegis
// to clean and validate user inputs before processing them.
//
// It shows how to use the sanitization module to clean and validate user inputs
// before processing them.
package main

import (
	"fmt"
	"log"

	"github.com/theinventorylib/aegis/core"
)

// This example demonstrates how to use the sanitization module in Aegis
// to clean and validate user inputs before processing them.

func main() {
	fmt.Println("=== Aegis Sanitization Examples ===")

	// Example 1: Sanitize user name input
	fmt.Println("1. Sanitizing User Names:")
	userName := "  John<script>alert('xss')</script>  Doe  "
	sanitized := core.SanitizeString(userName, nil)
	fmt.Printf("   Input:  %q\n", userName)
	fmt.Printf("   Output: %q\n\n", sanitized)

	// Example 2: Sanitize email addresses
	fmt.Println("2. Sanitizing Email Addresses:")
	email := "  John.Doe@EXAMPLE.COM  "
	sanitizedEmail := core.SanitizeEmail(email)
	fmt.Printf("   Input:  %q\n", email)
	fmt.Printf("   Output: %q\n\n", sanitizedEmail)

	// Example 3: Sanitize usernames
	fmt.Println("3. Sanitizing Usernames:")
	username := "John_Doe123!@#$%"
	sanitizedUsername := core.SanitizeUsername(username, 0)
	fmt.Printf("   Input:  %q\n", username)
	fmt.Printf("   Output: %q\n\n", sanitizedUsername)

	// Example 4: Sanitize URLs
	fmt.Println("4. Sanitizing URLs:")
	validURL := "https://example.com/path"
	dangerousURL := "javascript:alert('xss')"
	fmt.Printf("   Valid URL Input:     %q\n", validURL)
	fmt.Printf("   Valid URL Output:    %q\n", core.SanitizeURL(validURL))
	fmt.Printf("   Dangerous URL Input: %q\n", dangerousURL)
	fmt.Printf("   Dangerous URL Output: %q (blocked)\n\n", core.SanitizeURL(dangerousURL))

	// Example 5: Sanitize filenames
	fmt.Println("5. Sanitizing Filenames:")
	filename := "../../etc/passwd"
	sanitizedFilename := core.SanitizeFilename(filename)
	fmt.Printf("   Input:  %q\n", filename)
	fmt.Printf("   Output: %q\n\n", sanitizedFilename)

	// Example 6: Sanitize HTML content
	fmt.Println("6. Sanitizing HTML Content:")
	htmlContent := "<script>alert('xss')</script><p>Hello World</p>"
	sanitizedHTML := core.SanitizeHTML(htmlContent)
	fmt.Printf("   Input:  %q\n", htmlContent)
	fmt.Printf("   Output: %q\n\n", sanitizedHTML)

	// Example 7: Sanitize multiline text
	fmt.Println("7. Sanitizing Multiline Text:")
	multilineText := "Line 1\r\nLine 2<script>alert('xss')</script>\nLine 3"
	sanitizedMultiline := core.SanitizeMultiline(multilineText, 0)
	fmt.Printf("   Input:  %q\n", multilineText)
	fmt.Printf("   Output: %q\n\n", sanitizedMultiline)

	// Example 8: Custom sanitization configuration
	fmt.Println("8. Custom Sanitization Configuration:")
	customConfig := &core.SanitizationConfig{
		MaxLength:           20,
		AllowUnicode:        false,
		StripHTML:           true,
		NormalizeWhitespace: true,
		TrimWhitespace:      true,
	}
	longText := "This is a very long text with 世界 Unicode characters"
	sanitizedCustom := core.SanitizeString(longText, customConfig)
	fmt.Printf("   Input:  %q\n", longText)
	fmt.Printf("   Output: %q (limited to 20 chars, no Unicode)\n\n", sanitizedCustom)

	// Example 9: Phone number sanitization
	fmt.Println("9. Sanitizing Phone Numbers:")
	phone := "+1 (555) 123-4567"
	sanitizedPhone := core.SanitizePhoneNumber(phone)
	fmt.Printf("   Input:  %q\n", phone)
	fmt.Printf("   Output: %q\n\n", sanitizedPhone)

	// Example 10: Practical user registration flow
	fmt.Println("10. Practical User Registration Flow:")
	registerUser("  Alice  ", "  Alice@EXAMPLE.com  ", "alice_123!@#")
}

// registerUser demonstrates a practical use case of sanitization in a user registration flow
func registerUser(name, email, username string) {
	fmt.Println("    Registering user...")

	// Sanitize all inputs
	sanitizedName := core.SanitizeString(name, nil)
	sanitizedEmail := core.SanitizeEmail(email)
	sanitizedUsername := core.SanitizeUsername(username, 0)

	// Validate after sanitization
	if err := core.ValidateEmail(sanitizedEmail); err != nil {
		log.Printf("    ❌ Invalid email: %v\n", err)
		return
	}

	fmt.Printf("    ✓ Name:     %q (sanitized from %q)\n", sanitizedName, name)
	fmt.Printf("    ✓ Email:    %q (sanitized from %q)\n", sanitizedEmail, email)
	fmt.Printf("    ✓ Username: %q (sanitized from %q)\n", sanitizedUsername, username)
	fmt.Println("    ✓ User registration successful!")
}
