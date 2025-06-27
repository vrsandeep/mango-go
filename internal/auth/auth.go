package auth

import "golang.org/x/crypto/bcrypt"

// HashPassword generates a bcrypt hash of the password.
// The cost parameter (14) is a good balance between security and performance.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

// CheckPasswordHash compares a plaintext password with a stored bcrypt hash.
// It returns true if the password matches the hash.
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
