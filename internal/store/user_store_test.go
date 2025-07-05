package store_test

import (
	"testing"
	"time"

	"github.com/vrsandeep/mango-go/internal/auth"
	"github.com/vrsandeep/mango-go/internal/store"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

func TestUserStore_CreateAndGet(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

	passwordHash, _ := auth.HashPassword("password123")

	t.Run("Create User Success", func(t *testing.T) {
		user, err := s.CreateUser("testuser", passwordHash, "user")
		if err != nil {
			t.Fatalf("CreateUser failed: %v", err)
		}
		if user.Username != "testuser" {
			t.Errorf("Expected username 'testuser', got '%s'", user.Username)
		}
	})

	t.Run("Create User with Duplicate Username", func(t *testing.T) {
		_, err := s.CreateUser("testuser", passwordHash, "user")
		if err == nil {
			t.Fatal("Expected error when creating user with duplicate username, but got nil")
		}
	})

	t.Run("Get User By Username", func(t *testing.T) {
		user, err := s.GetUserByUsername("testuser")
		if err != nil {
			t.Fatalf("GetUserByUsername failed: %v", err)
		}
		if user.Username != "testuser" {
			t.Errorf("Expected username 'testuser', got '%s'", user.Username)
		}
		if !auth.CheckPasswordHash("password123", user.PasswordHash) {
			t.Error("Password hash does not match")
		}
	})

	t.Run("Get Non-existent User", func(t *testing.T) {
		_, err := s.GetUserByUsername("nonexistent")
		if err == nil {
			t.Fatal("Expected error when getting non-existent user, but got nil")
		}
	})
}

func TestUserStore_UpdateAndDelete(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

	passwordHash, _ := auth.HashPassword("password123")
	user, _ := s.CreateUser("userToUpdate", passwordHash, "user")

	t.Run("Update User Info", func(t *testing.T) {
		err := s.UpdateUser(user.ID, "updatedUsername", "admin")
		if err != nil {
			t.Fatalf("UpdateUser failed: %v", err)
		}
		updatedUser, _ := s.GetUserByID(user.ID)
		if updatedUser.Username != "updatedUsername" || updatedUser.Role != "admin" {
			t.Errorf("User info was not updated correctly. Got: %+v", updatedUser)
		}
	})

	t.Run("Update User Password", func(t *testing.T) {
		newPasswordHash, _ := auth.HashPassword("newpassword")
		err := s.UpdateUserPassword(user.ID, newPasswordHash)
		if err != nil {
			t.Fatalf("UpdateUserPassword failed: %v", err)
		}
		updatedUser, _ := s.GetUserByID(user.ID)
		if !auth.CheckPasswordHash("newpassword", updatedUser.PasswordHash) {
			t.Error("Password was not updated correctly")
		}
	})

	t.Run("Delete User", func(t *testing.T) {
		err := s.DeleteUser(user.ID)
		if err != nil {
			t.Fatalf("DeleteUser failed: %v", err)
		}
		_, err = s.GetUserByID(user.ID)
		if err == nil {
			t.Error("Expected error when getting deleted user, but got nil")
		}
	})
}

func TestUserStore_Sessions(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)
	passwordHash, _ := auth.HashPassword("password123")
	user, _ := s.CreateUser("sessionuser", passwordHash, "user")

	t.Run("Create and Get Session", func(t *testing.T) {
		token, err := s.CreateSession(user.ID)
		if err != nil {
			t.Fatalf("CreateSession failed: %v", err)
		}
		if token == "" {
			t.Fatal("CreateSession returned an empty token")
		}

		sessionUser, err := s.GetUserFromSession(token)
		if err != nil {
			t.Fatalf("GetUserFromSession failed: %v", err)
		}
		if sessionUser.ID != user.ID {
			t.Errorf("Session returned wrong user. Expected ID %d, got %d", user.ID, sessionUser.ID)
		}
	})

	t.Run("Get Expired Session", func(t *testing.T) {
		// Manually insert an expired token
		expiredToken := "expired-token"
		expiry := time.Now().Add(-1 * time.Hour)
		db.Exec("INSERT INTO sessions (token, user_id, expiry) VALUES (?, ?, ?)", expiredToken, user.ID, expiry)

		_, err := s.GetUserFromSession(expiredToken)
		if err == nil {
			t.Fatal("Expected error for expired session, but got nil")
		}
		if err.Error() != "session expired" {
			t.Errorf("Expected error message 'session expired', got '%s'", err.Error())
		}
	})

	t.Run("Delete Session", func(t *testing.T) {
		token, _ := s.CreateSession(user.ID)
		err := s.DeleteSession(token)
		if err != nil {
			t.Fatalf("DeleteSession failed: %v", err)
		}
		_, err = s.GetUserFromSession(token)
		if err == nil {
			t.Fatal("Expected error after deleting session, but got nil")
		}
	})
}

func TestUserStore_ListAndCount(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

	count, err := s.CountUsers()
	if err != nil {
		t.Fatalf("CountUsers failed on empty DB: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 users, got %d", count)
	}

	passwordHash, _ := auth.HashPassword("password123")
	s.CreateUser("user1", passwordHash, "user")
	s.CreateUser("user2", passwordHash, "admin")

	users, err := s.ListUsers()
	if err != nil {
		t.Fatalf("ListUsers failed: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("Expected to list 2 users, got %d", len(users))
	}

	count, err = s.CountUsers()
	if err != nil {
		t.Fatalf("CountUsers failed on populated DB: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 users, got %d", count)
	}
}
