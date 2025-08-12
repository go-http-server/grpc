package service

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/go-http-server/grpc/protoc"
	"github.com/jinzhu/copier"
)

var ErrAlreadyExists = errors.New("laptop already exists")

// LaptopStore defines the interface for storing laptops.
type LaptopStore interface {
	// Save persists a laptop to the storage.
	Save(laptop *protoc.Laptop) error

	// Find retrieves a laptop by its ID.
	Find(id string) (*protoc.Laptop, error)

	Search(ctx context.Context, filter *protoc.Filter, found func(laptop *protoc.Laptop) error) error
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
	other, err := deepCopyLaptop(laptop)
	if err != nil {
		return err
	}

	mem.laptops[laptop.Id] = other
	return nil
}

func (mem *InMemoryLaptopStore) Find(id string) (*protoc.Laptop, error) {
	mem.mu.RLock()
	defer mem.mu.RUnlock()

	laptop, ok := mem.laptops[id]
	if !ok {
		return nil, fmt.Errorf("laptop with id %s not found", id)
	}

	// deep copy the laptop to avoid external modifications
	return deepCopyLaptop(laptop)
}

func (mem *InMemoryLaptopStore) Search(ctx context.Context, filter *protoc.Filter, found func(laptop *protoc.Laptop) error) error {
	mem.mu.RLock()
	defer mem.mu.RUnlock()

	for _, laptop := range mem.laptops {
		err := contextError(ctx)
		if err != nil {
			return err
		}

		if isQualified(filter, laptop) {
			// deep copy the laptop to avoid external modifications
			other, err := deepCopyLaptop(laptop)
			if err != nil {
				return err
			}

			err = found(other)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func isQualified(filter *protoc.Filter, laptop *protoc.Laptop) bool {
	if laptop == nil || filter == nil {
		return false
	}

	if laptop.GetPriceUsd() > filter.GetMaxPriceUsd() {
		return false
	}

	if laptop.GetCpu().GetNumCores() < filter.GetMinCpuCores() {
		return false
	}

	if laptop.GetCpu().GetMinGhz() < filter.GetMinCpuGhz() {
		return false
	}

	if toBit(laptop.GetRam()) < toBit(filter.GetMinMemory()) {
		return false
	}

	return true
}

func toBit(memory *protoc.Memory) uint64 {
	value := memory.GetValue()

	switch memory.GetUnit() {
	case protoc.Memory_BIT:
		return value
	case protoc.Memory_BYTE:
		return value << 3
	case protoc.Memory_KILOBYTE:
		return value << 13
	case protoc.Memory_MEGABYTE:
		return value << 23
	case protoc.Memory_GIGABYTE:
		return value << 33
	case protoc.Memory_TERABYTE:
		return value << 43
	default:
		return 0
	}
}

func deepCopyLaptop(laptop *protoc.Laptop) (*protoc.Laptop, error) {
	if laptop == nil {
		return nil, nil
	}

	other := &protoc.Laptop{}
	err := copier.Copy(other, laptop)
	if err != nil {
		return nil, err
	}

	return other, nil
}
