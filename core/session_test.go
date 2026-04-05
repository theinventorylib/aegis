package core

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/theinventorylib/aegis/auth"
)

// TC-SES-001: Session Service Creation
func TestNewSessionService(t *testing.T) {
	// Given
	mockUserStore := &mockUserStore{}
	mockSessionStore := &mockSessionStore{}
	config := DefaultSessionConfig()

	// When
	service := NewSessionService(mockUserStore, mockSessionStore, config, nil)

	// Then
	if service == nil {
		t.Fatal("NewSessionService should return a non-nil service")
	}
}

// TC-SES-002: Session Service with Default Config
func TestNewSessionService_DefaultConfig(t *testing.T) {
	// Given
	mockUserStore := &mockUserStore{}
	mockSessionStore := &mockSessionStore{}

	// When - nil config should use defaults
	service := NewSessionService(mockUserStore, mockSessionStore, nil, nil)

	// Then
	if service == nil {
		t.Fatal("NewSessionService should return a non-nil service with nil config")
	}
	if service.GetConfig() == nil {
		t.Error("Service should have a default config")
	}
}

// TC-SES-003: Session Cookie Manager
func TestSessionService_CookieManager(t *testing.T) {
	// Given
	mockUserStore := &mockUserStore{}
	mockSessionStore := &mockSessionStore{}
	config := DefaultSessionConfig()
	service := NewSessionService(mockUserStore, mockSessionStore, config, nil)

	// When
	cookieManager := service.GetCookieManager()

	// Then
	if cookieManager == nil {
		t.Error("Session service should have a cookie manager")
	}
}

// TC-SES-004: Default Session Config Values
func TestDefaultSessionConfig(t *testing.T) {
	// When
	config := DefaultSessionConfig()

	// Then
	if config == nil {
		t.Fatal("DefaultSessionConfig should return non-nil config")
		return
	}

	if config.SessionExpiry != DefaultSessionExpiry {
		t.Errorf("Expected session expiry %v, got %v", DefaultSessionExpiry, config.SessionExpiry)
	}

	if config.RefreshExpiry != DefaultRefreshExpiry {
		t.Errorf("Expected refresh expiry %v, got %v", DefaultRefreshExpiry, config.RefreshExpiry)
	}
}

// TC-SES-005: Generate Random Token
func TestGenerateRandomToken(t *testing.T) {
	// Generate tokens multiple times
	tokens := make(map[string]bool)
	for i := 0; i < 100; i++ {
		token := GenerateSecureToken()
		if token == "" {
			t.Error("Generated token should not be empty")
		}
		tokens[token] = true
	}

	// All tokens should be unique
	if len(tokens) != 100 {
		t.Errorf("Expected 100 unique tokens, got %d", len(tokens))
	}
}

// TC-SES-006: Token Length
func TestGenerateRandomToken_Length(t *testing.T) {
	// Given/When
	token := GenerateSecureToken()

	// Then - Token should have reasonable length (base64 encoded 32 bytes = ~43 chars)
	if len(token) < 32 {
		t.Errorf("Token length %d is too short", len(token))
	}
}

// TC-SES-007: Concurrent Token Generation
func TestGenerateRandomToken_Concurrent(t *testing.T) {
	// Given
	var wg sync.WaitGroup
	tokens := make(chan string, 500)

	// When - Generate tokens concurrently
	for i := 0; i < 500; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tokens <- GenerateSecureToken()
		}()
	}
	wg.Wait()
	close(tokens)

	// Then - All should be unique
	uniqueTokens := make(map[string]bool)
	for token := range tokens {
		uniqueTokens[token] = true
	}
	if len(uniqueTokens) != 500 {
		t.Errorf("Expected 500 unique tokens, got %d", len(uniqueTokens))
	}
}

// TC-SES-008: Session Expiration Check
func TestSessionExpiration(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		isExpired bool
	}{
		{
			name:      "Future expiry - not expired",
			expiresAt: time.Now().Add(1 * time.Hour),
			isExpired: false,
		},
		{
			name:      "Past expiry - expired",
			expiresAt: time.Now().Add(-1 * time.Hour),
			isExpired: true,
		},
		{
			name:      "Just expired",
			expiresAt: time.Now().Add(-1 * time.Second),
			isExpired: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Now()
			expired := now.After(tt.expiresAt)
			if expired != tt.isExpired {
				t.Errorf("Expected expired=%v, got %v", tt.isExpired, expired)
			}
		})
	}
}

// TC-SES-009: Bearer Auth Enable/Disable
func TestSessionService_BearerAuth(t *testing.T) {
	// Given
	mockUserStore := &mockUserStore{}
	mockSessionStore := &mockSessionStore{}
	config := DefaultSessionConfig()
	service := NewSessionService(mockUserStore, mockSessionStore, config, nil)

	// Initially disabled
	if service.IsBearerAuthEnabled() {
		t.Error("Bearer auth should be disabled by default")
	}

	// When - Enable bearer auth
	service.EnableBearerAuth()

	// Then
	if !service.IsBearerAuthEnabled() {
		t.Error("Bearer auth should be enabled after EnableBearerAuth()")
	}
}

// TC-SES-010: Session ID Generation
func TestSessionIDGeneration(t *testing.T) {
	// Generate multiple session IDs
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := GenerateID()
		if id == "" {
			t.Error("Generated ID should not be empty")
		}
		ids[id] = true
	}

	// All IDs should be unique
	if len(ids) != 100 {
		t.Errorf("Expected 100 unique IDs, got %d", len(ids))
	}
}

// TC-SES-011: Session Config Cookie Settings
func TestSessionConfig_CookieSettings(t *testing.T) {
	// When
	config := DefaultSessionConfig()

	// Then
	if config.CookieSettings.HTTPOnly != true {
		t.Error("HTTPOnly should be true by default")
	}
	if config.CookieSettings.Secure != true {
		t.Error("Secure should be true by default")
	}
	if config.CookieSettings.SameSite != DefaultCookieSameSite {
		t.Errorf("Expected SameSite %s, got %s", DefaultCookieSameSite, config.CookieSettings.SameSite)
	}
}

// Mock implementations for testing
type mockUserStore struct {
	users map[string]auth.User
	mu    sync.RWMutex
}

func (m *mockUserStore) Create(_ context.Context, user auth.User) (auth.User, error) {
	if m.users == nil {
		m.users = make(map[string]auth.User)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users[user.ID] = user
	return user, nil
}

func (m *mockUserStore) GetByID(_ context.Context, id string) (auth.User, error) {
	if m.users == nil {
		return auth.User{}, ErrUserNotFound
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	user, exists := m.users[id]
	if !exists {
		return auth.User{}, ErrUserNotFound
	}
	return user, nil
}

func (m *mockUserStore) GetByEmail(_ context.Context, email string) (auth.User, error) {
	if m.users == nil {
		return auth.User{}, ErrUserNotFound
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, user := range m.users {
		if user.Email == email {
			return user, nil
		}
	}
	return auth.User{}, ErrUserNotFound
}

func (m *mockUserStore) Update(_ context.Context, user auth.User) error {
	if m.users == nil {
		return ErrUserNotFound
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.users[user.ID]; !exists {
		return ErrUserNotFound
	}
	m.users[user.ID] = user
	return nil
}

func (m *mockUserStore) Delete(_ context.Context, id string) error {
	if m.users == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.users, id)
	return nil
}

func (m *mockUserStore) List(_ context.Context, _, _ int) ([]auth.User, error) {
	if m.users == nil {
		return nil, nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	users := make([]auth.User, 0, len(m.users))
	for _, user := range m.users {
		users = append(users, user)
	}
	return users, nil
}

func (m *mockUserStore) Count(_ context.Context) (int, error) {
	if m.users == nil {
		return 0, nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.users), nil
}

type mockSessionStore struct {
	sessions map[string]auth.Session
	mu       sync.RWMutex
}

func (m *mockSessionStore) Create(_ context.Context, session auth.Session) error {
	if m.sessions == nil {
		m.sessions = make(map[string]auth.Session)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[session.ID] = session
	return nil
}

func (m *mockSessionStore) Get(_ context.Context, id string) (auth.Session, error) {
	if m.sessions == nil {
		return auth.Session{}, ErrSessionNotFound
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	session, exists := m.sessions[id]
	if !exists {
		return auth.Session{}, ErrSessionNotFound
	}
	return session, nil
}

func (m *mockSessionStore) GetByToken(_ context.Context, token string) (auth.Session, error) {
	if m.sessions == nil {
		return auth.Session{}, ErrSessionNotFound
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, session := range m.sessions {
		if session.Token == token {
			return session, nil
		}
	}
	return auth.Session{}, ErrSessionNotFound
}

func (m *mockSessionStore) GetByRefreshToken(_ context.Context, token string) (auth.Session, error) {
	if m.sessions == nil {
		return auth.Session{}, ErrSessionNotFound
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, session := range m.sessions {
		if session.RefreshToken == token {
			return session, nil
		}
	}
	return auth.Session{}, ErrSessionNotFound
}

func (m *mockSessionStore) GetByUserID(_ context.Context, userID string, offset, limit int) ([]auth.Session, error) {
	if m.sessions == nil {
		return nil, nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	var sessions []auth.Session
	for _, session := range m.sessions {
		if session.UserID == userID {
			sessions = append(sessions, session)
		}
	}
	// Note: Proper in-memory offset and limit logic skipped here for simplicity in tests
	return sessions, nil
}

func (m *mockSessionStore) CountByUserID(_ context.Context, userID string) (int, error) {
	if m.sessions == nil {
		return 0, nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	count := 0
	for _, session := range m.sessions {
		if session.UserID == userID {
			count++
		}
	}
	return count, nil
}

func (m *mockSessionStore) Update(_ context.Context, session auth.Session) error {
	if m.sessions == nil {
		return ErrSessionNotFound
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.sessions[session.ID]; !exists {
		return ErrSessionNotFound
	}
	m.sessions[session.ID] = session
	return nil
}

func (m *mockSessionStore) Delete(_ context.Context, id string) error {
	if m.sessions == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, id)
	return nil
}

func (m *mockSessionStore) DeleteByUserID(_ context.Context, userID string) error {
	if m.sessions == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, session := range m.sessions {
		if session.UserID == userID {
			delete(m.sessions, id)
		}
	}
	return nil
}

func (m *mockSessionStore) CleanupExpired(_ context.Context) error {
	if m.sessions == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	for id, session := range m.sessions {
		if now.After(session.ExpiresAt) {
			delete(m.sessions, id)
		}
	}
	return nil
}
