package emailotp

// Provider is the interface that email service providers must implement.
//
// Users should create their own implementation based on their email service
// (SMTP, SendGrid, Resend, AWS SES, Postmark, Mailgun, etc.).
//
// Abstraction Benefits:
//   - Swap email providers without changing plugin code
//   - Test with mock providers
//   - Use different providers for different environments
//
// Example Implementation (SMTP):
//
//	type SMTPProvider struct {
//	    host     string
//	    port     int
//	    username string
//	    password string
//	    from     string
//	}
//
//	func (p *SMTPProvider) SendOTP(to, code string) error {
//	    subject := "Your Verification Code"
//	    body := fmt.Sprintf("Your verification code is: %s\n\nThis code will expire in 10 minutes.", code)
//	    auth := smtp.PlainAuth("", p.username, p.password, p.host)
//	    msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s", p.from, to, subject, body)
//	    return smtp.SendMail(p.host+":"+strconv.Itoa(p.port), auth, p.from, []string{to}, []byte(msg))
//	}
//
//	func (p *SMTPProvider) VerifyOTP(to, code string) (bool, error) {
//	    // Verification logic handled by plugin's OTP storage
//	    // This method can be used for provider-specific validation if needed
//	    return true, nil
//	}
//
// Example Implementation (SendGrid):
//
//	type SendGridProvider struct {
//	    apiKey string
//	    from   string
//	}
//
//	func (p *SendGridProvider) SendOTP(to, code string) error {
//	    message := mail.NewV3Mail()
//	    message.SetFrom(mail.NewEmail("", p.from))
//	    message.AddContent(mail.NewContent("text/plain", fmt.Sprintf("Your OTP: %s", code)))
//	    personalization := mail.NewPersonalization()
//	    personalization.AddTos(mail.NewEmail("", to))
//	    message.AddPersonalizations(personalization)
//
//	    client := sendgrid.NewSendClient(p.apiKey)
//	    _, err := client.Send(message)
//	    return err
//	}
type Provider interface {
	// SendOTP sends a one-time password to the specified email address.
	//
	// Parameters:
	//   - to: Recipient email address
	//   - code: OTP code to send (e.g., "123456")
	//
	// Returns:
	//   - error: If email sending fails
	SendOTP(to, code string) error

	// VerifyOTP verifies an OTP code for an email address.
	//
	// Note: Most implementations delegate verification to the plugin's OTP storage.
	// This method is available for provider-specific validation if needed.
	//
	// Parameters:
	//   - to: Email address to verify
	//   - code: OTP code to verify
	//
	// Returns:
	//   - bool: true if OTP is valid
	//   - error: If verification fails
	VerifyOTP(to, code string) (bool, error)
}
