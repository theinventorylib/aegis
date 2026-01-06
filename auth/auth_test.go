package auth

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// Test constants for commonly used strings (goconst)
const (
	testEmail  = "test@example.com"
	testUserID = "user_123"
)

// Error definitions for tests
var (
	ErrUserNotFound    = errors.New("user not found")
	ErrAccountNotFound = errors.New("account not found")
	ErrSessionNotFound = errors.New("session not found")
	ErrDuplicateEmail  = errors.New("email already exists")
)

// =============================================================================
// Mock Store Implementations
// =============================================================================

// mockUserStore implements UserStore for testing
type mockUserStore struct {
	users map[string]User
	mu    sync.RWMutex
}

func newMockUserStore() *mockUserStore {
	return &mockUserStore{
		users: make(map[string]User),
	}
}

func (m *mockUserStore) Create(_ context.Context, user User) (User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, u := range m.users {
		if u.Email == user.Email && user.Email != "" {
			return User{}, ErrDuplicateEmail
		}
	}

	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now()
	}
	user.UpdatedAt = time.Now()
	m.users[user.ID] = user
	return user, nil
}

func (m *mockUserStore) GetByEmail(_ context.Context, email string) (User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, user := range m.users {
		if user.Email == email {
			return user, nil
		}
	}
	return User{}, ErrUserNotFound
}

func (m *mockUserStore) GetByID(_ context.Context, id string) (User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	user, exists := m.users[id]
	if !exists {
		return User{}, ErrUserNotFound
	}
	return user, nil
}

func (m *mockUserStore) Update(_ context.Context, user User) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.users[user.ID]; !exists {
		return ErrUserNotFound
	}
	user.UpdatedAt = time.Now()
	m.users[user.ID] = user
	return nil
}

func (m *mockUserStore) Delete(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.users[id]; !exists {
		return ErrUserNotFound
	}
	delete(m.users, id)
	return nil
}

func (m *mockUserStore) List(_ context.Context, offset, limit int) ([]User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	users := make([]User, 0, len(m.users))
	for _, user := range m.users {
		users = append(users, user)
	}
	if offset >= len(users) {
		return []User{}, nil
	}
	end := offset + limit
	if end > len(users) {
		end = len(users)
	}
	return users[offset:end], nil
}

func (m *mockUserStore) Count(_ context.Context) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.users), nil
}

// mockSessionStore implements SessionStore for testing
type mockSessionStore struct {
	sessions map[string]Session
	mu       sync.RWMutex
}

func newMockSessionStore() *mockSessionStore {
	return &mockSessionStore{
		sessions: make(map[string]Session),
	}
}

func (m *mockSessionStore) Create(_ context.Context, session Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if session.CreatedAt.IsZero() {
		session.CreatedAt = time.Now()
	}
	m.sessions[session.ID] = session
	return nil
}

func (m *mockSessionStore) Get(_ context.Context, id string) (Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	session, exists := m.sessions[id]
	if !exists {
		return Session{}, ErrSessionNotFound
	}
	return session, nil
}

func (m *mockSessionStore) GetByToken(_ context.Context, token string) (Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, session := range m.sessions {
		if session.Token == token {
			return session, nil
		}
	}
	return Session{}, ErrSessionNotFound
}

func (m *mockSessionStore) GetByRefreshToken(_ context.Context, refreshToken string) (Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, session := range m.sessions {
		if session.RefreshToken == refreshToken {
			return session, nil
		}
	}
	return Session{}, ErrSessionNotFound
}

func (m *mockSessionStore) GetByUserID(_ context.Context, userID string) ([]Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var sessions []Session
	for _, session := range m.sessions {
		if session.UserID == userID {
			sessions = append(sessions, session)
		}
	}
	return sessions, nil
}

func (m *mockSessionStore) Update(_ context.Context, session Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.sessions[session.ID]; !exists {
		return ErrSessionNotFound
	}
	m.sessions[session.ID] = session
	return nil
}

func (m *mockSessionStore) Delete(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, id)
	return nil
}

func (m *mockSessionStore) DeleteByUserID(_ context.Context, userID string) error {
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

// =============================================================================
// User Store Tests
// =============================================================================

func TestUserStore_Create(t *testing.T) {
	store := newMockUserStore()
	ctx := context.Background()
	user := User{ID: "user_123", Email: "test@example.com", Name: "Test User"}

	created, err := store.Create(ctx, user)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if created.ID != user.ID {
		t.Errorf("Expected ID %s, got %s", user.ID, created.ID)
	}

	retrieved, err := store.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if retrieved.Email != user.Email {
		t.Errorf("Expected email %s, got %s", user.Email, retrieved.Email)
	}
}

func TestUserStore_EmailUniqueness(t *testing.T) {
	store := newMockUserStore()
	ctx := context.Background()
	email := testEmail

	user1 := User{ID: "user_1", Email: email, Name: "User 1"}
	_, err := store.Create(ctx, user1)
	if err != nil {
		t.Fatalf("First create failed: %v", err)
	}

	user2 := User{ID: "user_2", Email: email, Name: "User 2"}
	_, err = store.Create(ctx, user2)
	if err == nil {
		t.Error("Expected error for duplicate email")
	}
	if !errors.Is(err, ErrDuplicateEmail) {
		t.Errorf("Expected ErrDuplicateEmail, got %v", err)
	}
}

func TestUserStore_GetByID(t *testing.T) {
	store := newMockUserStore()
	ctx := context.Background()
	user := User{ID: testUserID, Email: testEmail, Name: "Test"}
	if _, err := store.Create(ctx, user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	retrieved, err := store.GetByID(ctx, "user_123")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if retrieved.ID != user.ID {
		t.Errorf("Expected ID %s, got %s", user.ID, retrieved.ID)
	}
}

func TestUserStore_GetByID_NotFound(t *testing.T) {
	store := newMockUserStore()
	ctx := context.Background()

	_, err := store.GetByID(ctx, "nonexistent_user")
	if err == nil {
		t.Error("Expected error for nonexistent user")
	}
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}
}

func TestUserStore_GetByEmail(t *testing.T) {
	store := newMockUserStore()
	ctx := context.Background()
	user := User{ID: testUserID, Email: testEmail, Name: "Test"}
	if _, err := store.Create(ctx, user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	retrieved, err := store.GetByEmail(ctx, "test@example.com")
	if err != nil {
		t.Fatalf("GetByEmail failed: %v", err)
	}
	if retrieved.Email != user.Email {
		t.Errorf("Expected email %s, got %s", user.Email, retrieved.Email)
	}
}

func TestUserStore_GetByEmail_NotFound(t *testing.T) {
	store := newMockUserStore()
	ctx := context.Background()

	_, err := store.GetByEmail(ctx, "nonexistent@example.com")
	if err == nil {
		t.Error("Expected error for nonexistent email")
	}
}

func TestUserStore_Update(t *testing.T) {
	store := newMockUserStore()
	ctx := context.Background()
	user := User{ID: testUserID, Email: testEmail, Name: "Original Name"}
	if _, err := store.Create(ctx, user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	user.Name = "Updated Name"
	err := store.Update(ctx, user)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	retrieved, _ := store.GetByID(ctx, user.ID)
	if retrieved.Name != "Updated Name" {
		t.Errorf("Expected name 'Updated Name', got '%s'", retrieved.Name)
	}
}

func TestUserStore_Update_NotFound(t *testing.T) {
	store := newMockUserStore()
	ctx := context.Background()
	user := User{ID: "nonexistent", Email: "test@example.com"}

	err := store.Update(ctx, user)
	if err == nil {
		t.Error("Expected error for nonexistent user")
	}
}

func TestUserStore_Delete(t *testing.T) {
	store := newMockUserStore()
	ctx := context.Background()
	user := User{ID: "user_1", Email: "user1@example.com", Name: "User 1"}
	if _, err := store.Create(ctx, user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	err := store.Delete(ctx, user.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = store.GetByID(ctx, user.ID)
	if err == nil {
		t.Error("Expected error after deletion")
	}
}

func TestUserStore_List(t *testing.T) {
	store := newMockUserStore()
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		user := User{ID: string(rune('a' + i)), Email: string(rune('a'+i)) + "@example.com"}
		_, _ = store.Create(ctx, user)
	}

	users, err := store.List(ctx, 0, 2)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(users))
	}
}

func TestUserStore_Count(t *testing.T) {
	store := newMockUserStore()
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		user := User{ID: string(rune('a' + i)), Email: string(rune('a'+i)) + "@example.com"}
		_, _ = store.Create(ctx, user)
	}

	count, err := store.Count(ctx)
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 3 {
		t.Errorf("Expected count 3, got %d", count)
	}
}

func TestUserStore_ConcurrentAccess(t *testing.T) {
	store := newMockUserStore()
	ctx := context.Background()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			user := User{
				ID:    "user_" + string(rune(idx)),
				Email: "user" + string(rune(idx)) + "@example.com",
			}
			_, _ = store.Create(ctx, user)
		}(i)
	}
	wg.Wait()

	count, err := store.Count(ctx)
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count == 0 {
		t.Error("Expected some users to be created")
	}
}

// =============================================================================
// Session Store Tests
// =============================================================================

func TestSessionStore_Create(t *testing.T) {
	store := newMockSessionStore()
	ctx := context.Background()
	session := Session{
		ID:        "sess_123",
		UserID:    "user_123",
		Token:     "session_token",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	err := store.Create(ctx, session)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	retrieved, err := store.Get(ctx, session.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if retrieved.Token != session.Token {
		t.Errorf("Expected token %s, got %s", session.Token, retrieved.Token)
	}
}

func TestSessionStore_GetByToken(t *testing.T) {
	store := newMockSessionStore()
	ctx := context.Background()
	session := Session{
		ID:        "sess_123",
		UserID:    "user_123",
		Token:     "unique_session_token",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	_ = store.Create(ctx, session)

	retrieved, err := store.GetByToken(ctx, "unique_session_token")
	if err != nil {
		t.Fatalf("GetByToken failed: %v", err)
	}
	if retrieved.ID != session.ID {
		t.Errorf("Expected ID %s, got %s", session.ID, retrieved.ID)
	}
}

func TestSessionStore_GetByUserID(t *testing.T) {
	store := newMockSessionStore()
	ctx := context.Background()
	userID := testUserID

	_ = store.Create(ctx, Session{ID: "sess_1", UserID: userID, Token: "token1", ExpiresAt: time.Now().Add(time.Hour)})
	_ = store.Create(ctx, Session{ID: "sess_2", UserID: userID, Token: "token2", ExpiresAt: time.Now().Add(time.Hour)})
	_ = store.Create(ctx, Session{ID: "sess_3", UserID: "other_user", Token: "token3", ExpiresAt: time.Now().Add(time.Hour)})

	sessions, err := store.GetByUserID(ctx, userID)
	if err != nil {
		t.Fatalf("GetByUserID failed: %v", err)
	}
	if len(sessions) != 2 {
		t.Errorf("Expected 2 sessions, got %d", len(sessions))
	}
}

func TestSessionStore_DeleteByUserID(t *testing.T) {
	store := newMockSessionStore()
	ctx := context.Background()
	userID := testUserID

	_ = store.Create(ctx, Session{ID: "sess_1", UserID: userID, Token: "token1", ExpiresAt: time.Now().Add(time.Hour)})
	_ = store.Create(ctx, Session{ID: "sess_2", UserID: userID, Token: "token2", ExpiresAt: time.Now().Add(time.Hour)})
	_ = store.Create(ctx, Session{ID: "sess_3", UserID: "other_user", Token: "token3", ExpiresAt: time.Now().Add(time.Hour)})

	err := store.DeleteByUserID(ctx, userID)
	if err != nil {
		t.Fatalf("DeleteByUserID failed: %v", err)
	}

	sessions, _ := store.GetByUserID(ctx, userID)
	if len(sessions) != 0 {
		t.Errorf("Expected 0 sessions after deletion, got %d", len(sessions))
	}

	otherSessions, _ := store.GetByUserID(ctx, "other_user")
	if len(otherSessions) != 1 {
		t.Errorf("Expected 1 session for other user, got %d", len(otherSessions))
	}
}

func TestSessionStore_CleanupExpired(t *testing.T) {
	store := newMockSessionStore()
	ctx := context.Background()

	_ = store.Create(ctx, Session{ID: "expired_1", UserID: "user", Token: "t1", ExpiresAt: time.Now().Add(-1 * time.Hour)})
	_ = store.Create(ctx, Session{ID: "expired_2", UserID: "user", Token: "t2", ExpiresAt: time.Now().Add(-2 * time.Hour)})
	_ = store.Create(ctx, Session{ID: "valid_1", UserID: "user", Token: "t3", ExpiresAt: time.Now().Add(1 * time.Hour)})

	err := store.CleanupExpired(ctx)
	if err != nil {
		t.Fatalf("CleanupExpired failed: %v", err)
	}

	_, err = store.Get(ctx, "expired_1")
	if err == nil {
		t.Error("Expected expired_1 to be cleaned up")
	}

	_, err = store.Get(ctx, "valid_1")
	if err != nil {
		t.Error("Expected valid_1 to remain")
	}
}

func TestSessionStore_GetByRefreshToken(t *testing.T) {
	store := newMockSessionStore()
	ctx := context.Background()
	session := Session{
		ID:           "sess_123",
		UserID:       "user_123",
		Token:        "access_token",
		RefreshToken: "refresh_token_value",
		ExpiresAt:    time.Now().Add(24 * time.Hour),
	}
	_ = store.Create(ctx, session)

	retrieved, err := store.GetByRefreshToken(ctx, "refresh_token_value")
	if err != nil {
		t.Fatalf("GetByRefreshToken failed: %v", err)
	}
	if retrieved.ID != session.ID {
		t.Errorf("Expected ID %s, got %s", session.ID, retrieved.ID)
	}
}

// =============================================================================
// Model Tests
// =============================================================================

func TestUser_Methods(t *testing.T) {
	user := &User{}

	user.SetID("user_123")
	if user.GetID() != testUserID {
		t.Error("SetID/GetID failed")
	}

	user.SetEmail("test@example.com")
	if user.GetEmail() != "test@example.com" {
		t.Error("SetEmail/GetEmail failed")
	}

	user.SetName("Test User")
	if user.GetName() != "Test User" {
		t.Error("SetName/GetName failed")
	}

	now := time.Now()
	user.SetCreatedAt(now)
	user.SetUpdatedAt(now)
}

func TestAccount_Methods(t *testing.T) {
	account := &Account{}

	account.SetID("acc_123")
	if account.GetID() != "acc_123" {
		t.Error("SetID/GetID failed")
	}

	account.SetUserID("user_123")
	if account.GetUserID() != "user_123" {
		t.Error("SetUserID/GetUserID failed")
	}

	account.SetProvider("google")
	if account.GetProvider() != "google" {
		t.Error("SetProvider/GetProvider failed")
	}

	account.SetPasswordHash("hash123")
	if account.GetPasswordHash() != "hash123" {
		t.Error("SetPasswordHash/GetPasswordHash failed")
	}

	account.SetAccessToken("access_token")
	if account.GetAccessToken() != "access_token" {
		t.Error("SetAccessToken/GetAccessToken failed")
	}

	account.SetRefreshToken("refresh_token")
	if account.GetRefreshToken() != "refresh_token" {
		t.Error("SetRefreshToken/GetRefreshToken failed")
	}

	account.SetProviderAccountID("provider_123")
	if account.GetProviderAccountID() != "provider_123" {
		t.Error("SetProviderAccountID/GetProviderAccountID failed")
	}

	now := time.Now()
	account.SetExpiresAt(now)
	if account.GetExpiresAt() != now {
		t.Error("SetExpiresAt/GetExpiresAt failed")
	}

	account.SetCreatedAt(now)
	if account.GetCreatedAt() != now {
		t.Error("SetCreatedAt/GetCreatedAt failed")
	}

	account.SetUpdatedAt(now)
	if account.GetUpdatedAt() != now {
		t.Error("SetUpdatedAt/GetUpdatedAt failed")
	}
}

func TestSession_Methods(t *testing.T) {
	session := &Session{}

	session.SetID("sess_123")
	if session.GetID() != "sess_123" {
		t.Error("SetID/GetID failed")
	}

	session.SetUserID("user_123")
	if session.GetUserID() != "user_123" {
		t.Error("SetUserID/GetUserID failed")
	}

	session.SetToken("token_123")
	if session.GetToken() != "token_123" {
		t.Error("SetToken/GetToken failed")
	}

	session.SetRefreshToken("refresh_123")
	if session.GetRefreshToken() != "refresh_123" {
		t.Error("SetRefreshToken/GetRefreshToken failed")
	}

	now := time.Now()
	session.SetExpiresAt(now)
	if session.GetExpiresAt() != now {
		t.Error("SetExpiresAt/GetExpiresAt failed")
	}

	session.SetIPAddress("192.168.1.1")
	session.SetUserAgent("Mozilla/5.0")
	session.SetCreatedAt(now)
}

func TestVerification_Methods(t *testing.T) {
	v := &Verification{}

	v.SetID("ver_123")
	if v.GetID() != "ver_123" {
		t.Error("SetID/GetID failed")
	}

	v.SetToken("token_123")
	if v.GetToken() != "token_123" {
		t.Error("SetToken/GetToken failed")
	}

	v.SetIdentifier("test@example.com")
	if v.GetIdentifier() != "test@example.com" {
		t.Error("SetIdentifier/GetIdentifier failed")
	}

	now := time.Now()
	v.SetCreatedAt(now)
	if v.GetCreatedAt() != now {
		t.Error("SetCreatedAt/GetCreatedAt failed")
	}

	v.SetExpiresAt(now)
	if v.GetExpiresAt() != now {
		t.Error("SetExpiresAt/GetExpiresAt failed")
	}
}
