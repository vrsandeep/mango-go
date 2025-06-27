package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"

	"github.com/vrsandeep/mango-go/internal/models"
)

// ListUsers retrieves all users from the database, ordered by username.
func (s *Store) ListUsers() ([]*models.User, error) {
	rows, err := s.db.Query("SELECT id, username, role, created_at FROM users ORDER BY username ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var user models.User
		if err := rows.Scan(&user.ID, &user.Username, &user.Role, &user.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, &user)
	}
	return users, nil
}

// CreateUser adds a new user to the database.
func (s *Store) CreateUser(username, passwordHash, role string) (*models.User, error) {
	query := "INSERT INTO users (username, password_hash, role, created_at) VALUES (?, ?, ?, ?)"
	res, err := s.db.Exec(query, username, passwordHash, role, time.Now())
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &models.User{
		ID:       id,
		Username: username,
		Role:     role,
	}, nil
}

// UpdateUser updates a user's username and role.
func (s *Store) UpdateUser(id int64, username, role string) error {
	query := "UPDATE users SET username = ?, role = ? WHERE id = ?"
	_, err := s.db.Exec(query, username, role, id)
	return err
}

// UpdateUserPassword updates only the user's password hash.
func (s *Store) UpdateUserPassword(id int64, passwordHash string) error {
	query := "UPDATE users SET password_hash = ? WHERE id = ?"
	_, err := s.db.Exec(query, passwordHash, id)
	return err
}

// DeleteUser removes a user from the database. Cascading deletes will handle their sessions.
func (s *Store) DeleteUser(id int64) error {
	_, err := s.db.Exec("DELETE FROM users WHERE id = ?", id)
	return err
}

// GetUserByUsername retrieves a user by their unique username.
func (s *Store) GetUserByUsername(username string) (*models.User, error) {
	var user models.User
	query := "SELECT id, username, password_hash, role, created_at FROM users WHERE username = ?"
	err := s.db.QueryRow(query, username).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Role, &user.CreatedAt)
	return &user, err
}

// GetUserByID retrieves a user by their primary key.
func (s *Store) GetUserByID(id int64) (*models.User, error) {
	var user models.User
	query := "SELECT id, username, password_hash, role, created_at FROM users WHERE id = ?"
	err := s.db.QueryRow(query, id).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Role, &user.CreatedAt)
	return &user, err
}

// GetUserFromSession retrieves a user based on a session token.
func (s *Store) GetUserFromSession(token string) (*models.User, error) {
	var userID int64
	var expiry time.Time
	query := "SELECT user_id, expiry FROM sessions WHERE token = ?"
	err := s.db.QueryRow(query, token).Scan(&userID, &expiry)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("invalid session token")
		}
		return nil, err
	}

	if time.Now().After(expiry) {
		s.DeleteSession(token) // Clean up expired session
		return nil, errors.New("session expired")
	}

	return s.GetUserByID(userID)
}

// CountUsers returns the total number of users in the database.
func (s *Store) CountUsers() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	return count, err
}

// CreateSession creates a new session for a user and returns the session token.
func (s *Store) CreateSession(userID int64) (string, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", err
	}
	token := hex.EncodeToString(tokenBytes)
	expiry := time.Now().Add(7 * 24 * time.Hour) // 1 week session
	_, err := s.db.Exec("INSERT INTO sessions (token, user_id, expiry) VALUES (?, ?, ?)", token, userID, expiry)
	return token, err
}

// DeleteSession removes a session from the database (used for logout).
func (s *Store) DeleteSession(token string) error {
	_, err := s.db.Exec("DELETE FROM sessions WHERE token = ?", token)
	return err
}
