package service

import (
	"fmt"
	"sync"
)

type AccountStore interface {
	// Save persists the account to the store.
	Save(account *Account) error

	// Find retrieves an account by its username.
	Find(username string) (*Account, error)
}

// InMemoryAccountStore is an in-memory implementation of the AccountStore interface.
type InMemoryAccountStore struct {
	mutex    sync.RWMutex
	accounts map[string]*Account
}

func NewInMemoryAccountStore() AccountStore {
	return &InMemoryAccountStore{
		accounts: make(map[string]*Account),
	}
}

func (acc *InMemoryAccountStore) Save(account *Account) error {
	acc.mutex.Lock()
	defer acc.mutex.Unlock()

	// check if the account already exists
	if acc.accounts[account.Username] != nil {
		return fmt.Errorf("account with username %s already exists", account.Username)
	}

	// save clone account to avoid modifying the original
	acc.accounts[account.Username] = account.Clone()
	return nil
}

func (acc *InMemoryAccountStore) Find(username string) (*Account, error) {
	acc.mutex.RLock()
	defer acc.mutex.RUnlock()

	if account, ok := acc.accounts[username]; ok {
		return account.Clone(), nil
	}

	return nil, fmt.Errorf("account with username %s not found", username)
}
