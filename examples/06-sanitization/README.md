# Sanitization Example

This example demonstrates how to use Aegis's sanitization module to clean and validate user inputs.

## Overview

The sanitization module provides comprehensive input cleaning functions to prevent:
- **XSS (Cross-Site Scripting)** attacks
- **SQL Injection** attacks
- **Directory Traversal** attacks
- **Null Byte Injection**
- **Control Character Injection**

## Features Demonstrated

1. **General String Sanitization** - Remove HTML tags, normalize whitespace, enforce length limits
2. **Email Sanitization** - Normalize email addresses to lowercase, remove whitespace
3. **Username Sanitization** - Allow only safe characters, enforce naming conventions
4. **URL Sanitization** - Block dangerous URL schemes (javascript:, data:, etc.)
5. **Filename Sanitization** - Prevent directory traversal and dangerous characters
6. **HTML Content Sanitization** - Escape HTML entities to prevent XSS
7. **Multiline Text Sanitization** - Clean text areas while preserving formatting
8. **Custom Configuration** - Configure sanitization behavior for specific use cases
9. **Phone Number Sanitization** - Clean phone numbers to consistent format
10. **Practical Integration** - Real-world user registration flow example

## Running the Example

```bash
go run examples/06-sanitization/main.go
```

## Usage in Your Application

### Basic Usage

```go
import "github.com/theinventorylib/aegis/core"

// Sanitize user input
name := core.SanitizeString(userInput, nil)

// Sanitize email
email := core.SanitizeEmail(emailInput)

// Sanitize username
username := core.SanitizeUsername(usernameInput, 50)
```

### Custom Configuration

```go
config := &core.SanitizationConfig{
    MaxLength:           100,
    AllowUnicode:        true,
    StripHTML:           true,
    NormalizeWhitespace: true,
    TrimWhitespace:      true,
}

sanitized := core.SanitizeString(input, config)
```

### Integration with Validation

Always sanitize **before** validation:

```go
// 1. Sanitize first
email := core.SanitizeEmail(rawEmail)

// 2. Then validate
if err := core.ValidateEmail(email); err != nil {
    return fmt.Errorf("invalid email: %w", err)
}

// 3. Now safe to use
user.Email = email
```

## Security Best Practices

1. **Defense in Depth** - Sanitization is ONE layer of security. Also use:
   - Parameterized queries for SQL
   - Content Security Policy (CSP) headers
   - Output encoding in templates
   - Input validation

2. **Sanitize Early** - Clean inputs as soon as they enter your system

3. **Validate After Sanitization** - Ensure sanitized data meets your requirements

4. **Context-Specific Sanitization** - Use the right function for each input type:
   - `SanitizeEmail()` for emails
   - `SanitizeUsername()` for usernames
   - `SanitizeURL()` for URLs
   - `SanitizeFilename()` for file uploads
   - `SanitizeMultiline()` for text areas

5. **Never Trust User Input** - Always sanitize, even from authenticated users

## Available Functions

| Function | Purpose | Example |
|----------|---------|---------|
| `SanitizeString()` | General text cleaning | User names, descriptions |
| `SanitizeEmail()` | Email normalization | Email addresses |
| `SanitizeUsername()` | Username cleaning | Login usernames |
| `SanitizeURL()` | URL validation | Profile URLs, links |
| `SanitizeFilename()` | File upload safety | Uploaded filenames |
| `SanitizeHTML()` | HTML escaping | User-generated content |
| `SanitizeSQL()` | SQL pattern removal | Defense-in-depth only |
| `SanitizePhoneNumber()` | Phone formatting | Phone numbers |
| `SanitizeMultiline()` | Text area cleaning | Comments, descriptions |
| `StripTags()` | Quick HTML removal | Simple text extraction |
| `NormalizeWhitespace()` | Whitespace cleanup | Text formatting |

## Common Patterns

### User Registration

```go
func RegisterUser(name, email, username, password string) error {
    // Sanitize inputs
    name = core.SanitizeString(name, nil)
    email = core.SanitizeEmail(email)
    username = core.SanitizeUsername(username, 50)
    
    // Validate
    if err := core.ValidateEmail(email); err != nil {
        return err
    }
    if err := core.ValidatePassword(password, nil); err != nil {
        return err
    }
    
    // Create user...
    return nil
}
```

### File Upload

```go
func HandleFileUpload(filename string, content []byte) error {
    // Sanitize filename to prevent directory traversal
    safeFilename := core.SanitizeFilename(filename)
    
    // Ensure filename is not empty after sanitization
    if safeFilename == "" {
        return errors.New("invalid filename")
    }
    
    // Save file with sanitized name
    return os.WriteFile(safeFilename, content, 0644)
}
```

### Comment System

```go
func CreateComment(userID, content string) error {
    // Sanitize multiline content
    content = core.SanitizeMultiline(content, 5000)
    
    // Ensure content is not empty
    if strings.TrimSpace(content) == "" {
        return errors.New("comment cannot be empty")
    }
    
    // Store comment...
    return nil
}
```

## Learn More

- See `core/sanitization.go` for implementation details
- See `core/sanitization_test.go` for comprehensive test cases
- See `core/validation.go` for validation functions
