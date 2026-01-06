package sms

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/nyaruka/phonenumbers"
)

// Provider is the interface that SMS service providers must implement.
//
// Users should create their own implementation based on their SMS service
// (Twilio, AWS SNS, Vonage, MessageBird, Plivo, etc.).
//
// Abstraction Benefits:
//   - Swap SMS providers without changing plugin code
//   - Test with mock providers
//   - Use different providers for different regions
//
// Example Implementation (Twilio):
//
//	type TwilioProvider struct {
//	    accountSID string
//	    authToken  string
//	    from       string  // Twilio phone number
//	}
//
//	func (p *TwilioProvider) SendOTP(to, code string) error {
//	    msgData := url.Values{}
//	    msgData.Set("To", to)
//	    msgData.Set("From", p.from)
//	    msgData.Set("Body", fmt.Sprintf("Your verification code is: %s", code))
//
//	    urlStr := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", p.accountSID)
//	    req, _ := http.NewRequest("POST", urlStr, strings.NewReader(msgData.Encode()))
//	    req.SetBasicAuth(p.accountSID, p.authToken)
//	    req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
//
//	    client := &http.Client{}
//	    resp, err := client.Do(req)
//	    return err
//	}
//
//	func (p *TwilioProvider) VerifyOTP(to, code string) (bool, error) {
//	    return true, nil  // Verification handled by plugin
//	}
//
// Example Implementation (AWS SNS):
//
//	type SNSProvider struct {
//	    client *sns.Client
//	}
//
//	func (p *SNSProvider) SendOTP(to, code string) error {
//	    _, err := p.client.Publish(context.TODO(), &sns.PublishInput{
//	        PhoneNumber: aws.String(to),
//	        Message:     aws.String(fmt.Sprintf("Your OTP: %s", code)),
//	    })
//	    return err
//	}
type Provider interface {
	// SendOTP sends a one-time password to the specified phone number.
	//
	// Parameters:
	//   - to: Recipient phone number in E.164 format (e.g., "+14155551234")
	//   - code: OTP code to send (e.g., "123456")
	//
	// Returns:
	//   - error: If SMS sending fails
	SendOTP(to, code string) error

	// VerifyOTP verifies an OTP code for a phone number.
	//
	// Note: Most implementations delegate verification to the plugin's OTP storage.
	// This method is available for provider-specific validation if needed.
	//
	// Parameters:
	//   - to: Phone number to verify
	//   - code: OTP code to verify
	//
	// Returns:
	//   - bool: true if OTP is valid
	//   - error: If verification fails
	VerifyOTP(to, code string) (bool, error)
}

// ValidatePhoneNumber validates a phone number format using libphonenumber.
//
// This function performs comprehensive phone number validation:
//   - Basic format check (regex)
//   - International format validation (E.164)
//   - Country code verification
//   - Number length validation
//
// Parameters:
//   - phone: Phone number to validate (can include country code prefix)
//
// Returns:
//   - error: If phone number is empty or invalid
//
// Example:
//
//	err := ValidatePhoneNumber("+14155551234")  // Valid US number
//	err := ValidatePhoneNumber("+442071838750") // Valid UK number
func ValidatePhoneNumber(phone string) error {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return fmt.Errorf("phone number is required")
	}

	// First, try basic regex validation for obvious invalid formats
	if !regexp.MustCompile(`^\+?[\d\s\-\(\)]+$`).MatchString(phone) {
		return fmt.Errorf("invalid phone number format")
	}

	// Use libphonenumber for comprehensive validation
	num, err := phonenumbers.Parse(phone, "")
	if err != nil {
		return fmt.Errorf("invalid phone number: %w", err)
	}

	if !phonenumbers.IsValidNumber(num) {
		return fmt.Errorf("invalid phone number")
	}

	return nil
}
