package service

import (
	"errors"
	"sync"

	"github.com/go-http-server/grpc/protoc"
	"github.com/jinzhu/copier"
)

var ErrAlreadyExists = errors.New("laptop already exists")

// LaptopStore defines the interface for storing laptops.
type LaptopStore interface {
	// Save persists a laptop to the storage.
	Save(laptop *protoc.Laptop) error
}

// InMemoryLaptopStore is an in-memory implementation of LaptopStore.
type InMemoryLaptopStore struct {
	mu      sync.RWMutex
	laptops map[string]*protoc.Laptop
}

// NewInMemoryLaptopStore creates a new instance of InMemoryLaptopStore.
func NewInMemoryLaptopStore() *InMemoryLaptopStore {
	return &InMemoryLaptopStore{
		laptops: make(map[string]*protoc.Laptop),
	}
}

func (mem *InMemoryLaptopStore) Save(laptop *protoc.Laptop) error {
	mem.mu.Lock()
	defer mem.mu.Unlock()

	if mem.laptops[laptop.Id] != nil {
		return ErrAlreadyExists
	}

	// deep copy the laptop to avoid external modifications
	other := &protoc.Laptop{}
	err := copier.Copy(other, laptop)
	if err != nil {
		return err
	}

	mem.laptops[laptop.Id] = other
	return nil
}
