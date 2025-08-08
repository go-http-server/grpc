package service

import (
	"golang.org/x/crypto/bcrypt"
)

// Account represents a user account in the system.
type Account struct {
	Username, HashedPassword, Role string
}

// NewAccount creates a new account with the given username, password, and role.
func NewAccount(username, password, role string) (*Account, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	return &Account{
		Username:       username,
		HashedPassword: string(hashed),
		Role:           role,
	}, nil
}

// IsCorrectPassword checks if the provided password matches the account's hashed password.
func (acc *Account) IsCorrectPassword(password string) error {
	return bcrypt.CompareHashAndPassword([]byte(acc.HashedPassword), []byte(password))
}

// Clone creates a copy of the account.
func (acc *Account) Clone() *Account {
	return &Account{
		Username:       acc.Username,
		HashedPassword: acc.HashedPassword,
		Role:           acc.Role,
	}
}
