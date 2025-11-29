package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/theinventorylib/aegis/db"
	"github.com/theinventorylib/aegis/models"
)

// MockDB implements DBProvider for testing
type MockDB struct {
	mu           sync.RWMutex // Protects users and sessions maps
	users        map[string]*models.User
	sessions     map[string]*models.Session
	ExecFunc     func(context.Context, string, ...interface{}) (db.Result, error)
	QueryFunc    func(context.Context, string, ...interface{}) (db.Rows, error)
	QueryRowFunc func(context.Context, string, ...interface{}) db.Row
}

// NewMockDB creates a new mock database for testing.
func NewMockDB() *MockDB {
	return &MockDB{
		users:    make(map[string]*models.User),
		sessions: make(map[string]*models.Session),
	}
}

// CreateUser creates a test user.
func (m *MockDB) CreateUser(_ context.Context) (*models.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	user := &models.User{
		ID:        fmt.Sprintf("test-user-%d", len(m.users)),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.users[user.ID] = user
	return user, nil
}

// GetUserByID retrieves a user by ID.
func (m *MockDB) GetUserByID(_ context.Context, id string) (*models.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if u, ok := m.users[id]; ok {
		return u, nil
	}
	return nil, fmt.Errorf("user not found")
}

// UpdateUser updates a user.
func (m *MockDB) UpdateUser(_ context.Context, user *models.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.users[user.ID] = user
	return nil
}

// DeleteUser deletes a user.
func (m *MockDB) DeleteUser(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.users, id)
	return nil
}

// ListUsers returns a list of users.
func (m *MockDB) ListUsers(_ context.Context, _, _ int) ([]*models.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	users := make([]*models.User, 0, len(m.users))
	for _, u := range m.users {
		users = append(users, u)
	}
	return users, nil
}

// CountUsers returns the total number of users.
func (m *MockDB) CountUsers(_ context.Context) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.users), nil
}

// CreateSession creates a mock session.
func (m *MockDB) CreateSession(_ context.Context, session *models.Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.sessions[session.Token] = session
	return nil
}

// GetSession retrieves a session by token.
func (m *MockDB) GetSession(_ context.Context, token string) (*models.Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if s, ok := m.sessions[token]; ok {
		// Return a copy to avoid mutation issues
		sessionCopy := *s
		return &sessionCopy, nil
	}
	return nil, fmt.Errorf("session not found")
}

// GetSessionByRefreshToken retrieves a session by refresh token.
func (m *MockDB) GetSessionByRefreshToken(_ context.Context, refreshToken string) (*models.Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, s := range m.sessions {
		if s.RefreshToken == refreshToken {
			// Return a copy to avoid mutation issues
			sessionCopy := *s
			return &sessionCopy, nil
		}
	}
	return nil, fmt.Errorf("session not found")
}

// UpdateSession updates a session.
func (m *MockDB) UpdateSession(_ context.Context, session *models.Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// When updating a session, we need to remove the old entry and add the new one
	// because the token might have changed
	for oldToken, oldSession := range m.sessions {
		if oldSession.ID == session.ID {
			delete(m.sessions, oldToken)
			break
		}
	}
	m.sessions[session.Token] = session
	return nil
}

// DeleteSession deletes a session by token.
func (m *MockDB) DeleteSession(_ context.Context, token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.sessions, token)
	return nil
}

// GetUserSessions returns all sessions for a user.
func (m *MockDB) GetUserSessions(_ context.Context, _ string) ([]*models.Session, error) {
	return nil, nil
}

// DeleteUserSessions deletes all sessions for a user.
func (m *MockDB) DeleteUserSessions(_ context.Context, _ string) error {
	return nil
}

// Query implements db.Provider.
func (m *MockDB) Query(ctx context.Context, query string, args ...interface{}) (db.Rows, error) {
	if m.QueryFunc != nil {
		return m.QueryFunc(ctx, query, args...)
	}
	return nil, fmt.Errorf("not implemented in mock")
}

// QueryRow implements db.Provider.
func (m *MockDB) QueryRow(ctx context.Context, query string, args ...interface{}) db.Row {
	if m.QueryRowFunc != nil {
		return m.QueryRowFunc(ctx, query, args...)
	}
	return &mockRow{err: fmt.Errorf("not implemented in mock")}
}

// Exec implements db.Provider.
func (m *MockDB) Exec(ctx context.Context, query string, args ...interface{}) (db.Result, error) {
	if m.ExecFunc != nil {
		return m.ExecFunc(ctx, query, args...)
	}
	return nil, fmt.Errorf("not implemented in mock")
}

// Begin implements db.Provider.
func (m *MockDB) Begin(_ context.Context) (db.Tx, error) {
	return nil, fmt.Errorf("not implemented in mock")
}

// Close implements db.Provider
func (m *MockDB) Close() error {
	return nil
}

type mockRow struct {
	err error
}

func (r *mockRow) Scan(_ ...interface{}) error {
	return r.err
}
