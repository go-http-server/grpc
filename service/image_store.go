package service

import (
	"bytes"
	"fmt"
	"os"
	"sync"

	"github.com/google/uuid"
)

// ImageStore defines the interface for image storage operations.
type ImageStore interface {
	// Save stores an image for a laptop and returns the image ID and maybe have an error.
	Save(laptopID string, imageType string, imageData bytes.Buffer) (string, error)
}

// DiskImageStore implements the ImageStore interface, storing images on disk.
type DiskImageStore struct {
	mutex       sync.Mutex
	imageFolder string
	images      map[string]*MapInfo
}

// MapInfo holds information about the image associated with a laptop.
type MapInfo struct {
	LaptopID string
	Type     string
	Path     string
}

// NewDiskImageStore creates a new DiskImageStore with the specified image folder.
func NewDiskImageStore(imageFolder string) *DiskImageStore {
	return &DiskImageStore{imageFolder: imageFolder, images: make(map[string]*MapInfo)}
}

func (store *DiskImageStore) Save(laptopID string, imageType string, imageData bytes.Buffer) (string, error) {
	imgID, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}

	imagePath := fmt.Sprintf("%s/%s%s", store.imageFolder, imgID.String(), imageType)

	file, err := os.Create(imagePath)
	if err != nil {
		return "", err
	}

	_, err = imageData.WriteTo(file)
	if err != nil {
		return "", err
	}

	store.mutex.Lock()
	defer store.mutex.Unlock()

	store.images[imgID.String()] = &MapInfo{LaptopID: laptopID, Type: imageType, Path: imagePath}
	return imgID.String(), nil
}
