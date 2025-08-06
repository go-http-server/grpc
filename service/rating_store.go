package service

import "sync"

// RatingStore defines the interface for storing laptop ratings.
type RatingStore interface {
	// AddRating adds a rating for a laptop and returns the updated rating.
	AddRating(laptopID string, rating float64) (*Rating, error)
}

// Rating represents the rating of a laptop.
type Rating struct {
	Count uint32
	Sum   float64
}

// InMemoryRatingStore is an in-memory implementation of the RatingStore interface.
type InMemoryRatingStore struct {
	mutex   sync.RWMutex
	ratings map[string]*Rating
}

func NewInMemoryRatingStore() RatingStore {
	return &InMemoryRatingStore{
		ratings: make(map[string]*Rating),
	}
}

func (store *InMemoryRatingStore) AddRating(laptopID string, score float64) (*Rating, error) {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	rating := store.ratings[laptopID]
	if rating == nil {
		rating = &Rating{
			Count: 1,
			Sum:   score,
		}
	} else {
		rating.Count++
		rating.Sum += score
	}

	store.ratings[laptopID] = rating

	return rating, nil
}
