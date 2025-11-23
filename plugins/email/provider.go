package email

// Provider is the interface that email providers must implement.
// Users should create their own implementation based on their email service
// (SMTP, SendGrid, Resend, AWS SES, etc.)
//
// Example implementation with SMTP:
//
//	type SMTPProvider struct {
//	    host     string
//	    port     int
//	    username string
//	    password string
//	    from     string
//	}
//
//	func (p *SMTPProvider) SendEmail(to, subject, body string) error {
//	    // Use net/smtp or a library like gomail
//	    auth := smtp.PlainAuth("", p.username, p.password, p.host)
//	    msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s", p.from, to, subject, body)
//	    return smtp.SendMail(p.host+":"+strconv.Itoa(p.port), auth, p.from, []string{to}, []byte(msg))
//	}
//
//	func (p *SMTPProvider) SendOTP(to, code string) error {
//	    subject := "Your Verification Code"
//	    body := fmt.Sprintf("Your verification code is: %s\n\nThis code will expire in 10 minutes.", code)
//	    return p.SendEmail(to, subject, body)
//	}
type Provider interface {
	// SendEmail sends a generic email
	SendEmail(to, subject, body string) error

	// SendOTP sends an OTP verification code via email
	SendOTP(to, code string) error
}
