package oauth

import (
	"testing"

	"github.com/markbates/goth"
)

func TestProviderVerifiedEmail(t *testing.T) {
	tests := []struct {
		name string
		user goth.User
		want bool
	}{
		{"oidc bool true", goth.User{RawData: map[string]any{"email_verified": true}}, true},
		{"oidc bool false", goth.User{RawData: map[string]any{"email_verified": false}}, false},
		{"oidc string true", goth.User{RawData: map[string]any{"email_verified": "true"}}, true},
		{"oidc number nonzero", goth.User{RawData: map[string]any{"email_verified": float64(1)}}, true},
		{"github email trusted without flag", goth.User{Provider: "github", Email: "a@b.com"}, true},
		{"github without email", goth.User{Provider: "github"}, false},
		{"github explicit false still wins", goth.User{Provider: "github", Email: "a@b.com", RawData: map[string]any{"email_verified": false}}, false},
		{"unknown provider without flag", goth.User{Provider: "apple", Email: "a@b.com"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := providerVerifiedEmail(tt.user); got != tt.want {
				t.Errorf("providerVerifiedEmail() = %v, want %v", got, tt.want)
			}
		})
	}
}
