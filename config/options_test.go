package config

import (
	"testing"
	"time"

	"github.com/theinventorylib/aegis/v2/core"
)

func TestWithLoginAttemptConfigEnablesLockoutWithoutRateLimiter(t *testing.T) {
	cfg := Default()
	lac := &core.LoginAttemptConfig{
		MaxAttempts:     3,
		LockoutDuration: time.Minute,
		AttemptWindow:   time.Minute,
	}

	got := cfg.WithLoginAttemptConfig(lac)
	if got.LoginAttemptConfig != lac {
		t.Fatal("WithLoginAttemptConfig should assign the supplied config")
	}
	if got.RateLimitEnabled {
		t.Fatal("WithLoginAttemptConfig must not enable the HTTP rate limiter")
	}
}
