// Package sample provides a sample implementation of a keyboard generator.
package sample

import (
	"github.com/go-http-server/grpc/protoc"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func NewKeyBoard() *protoc.Keyboard {
	keyboard := &protoc.Keyboard{
		Layout:  randomKeyboardLayout(),
		Backlit: randomBool(),
	}

	return keyboard
}

func NewCPU() *protoc.CPU {
	brand := randomCPUBrand()
	name := randomCPUName(brand)

	cpu := &protoc.CPU{
		Brand:      brand,
		Name:       name,
		NumCores:   uint32(randomInt(4, 16)),
		NumThreads: uint32(randomInt(1, 4)),
		MinGhz:     randomFloat64(1.5, 2.5),
		MaxGhz:     randomFloat64(3.5, 6.5),
	}

	return cpu
}

func NewGPU() *protoc.GPU {
	brand := randomGPUBrand()
	name := randomGPUName(brand)
	memory := NewMemory()

	gpu := &protoc.GPU{
		Brand:  brand,
		Name:   name,
		Memory: memory,
		MinGhz: randomFloat64(1.5, 2.5),
		MaxGhz: randomFloat64(3.5, 6.5),
	}

	return gpu
}

func NewSSD() *protoc.Storage {
	ssd := &protoc.Storage{
		Driver: protoc.Storage_SSD,
		Memory: &protoc.Memory{
			Value: uint64(randomInt(128, 1024)),
			Unit:  protoc.Memory_GIGABYTE,
		},
	}

	return ssd
}

func NewHDD() *protoc.Storage {
	hdd := &protoc.Storage{
		Driver: protoc.Storage_HDD,
		Memory: &protoc.Memory{
			Value: uint64(randomInt(128, 1024)),
			Unit:  protoc.Memory_GIGABYTE,
		},
	}

	return hdd
}

func NewScreen() *protoc.Screen {
	screen := &protoc.Screen{
		Resolution: randomScreenResolution(),
		Size:       randomFloat32(13, 17),
		Panel:      randomScreenPanel(),
		Multitouch: randomBool(),
	}

	return screen
}

func NewLaptop() *protoc.Laptop {
	laptopBrand := randomLaptopBrand()
	laptopName := randomLaptopName(laptopBrand)

	laptop := &protoc.Laptop{
		Id:          uuid.NewString(),
		Brand:       laptopBrand,
		Name:        laptopName,
		Cpu:         NewCPU(),
		Ram:         NewMemory(),
		Gpus:        []*protoc.GPU{NewGPU(), NewGPU()},
		Storages:    []*protoc.Storage{NewSSD(), NewHDD()},
		Screen:      NewScreen(),
		Keyboard:    NewKeyBoard(),
		Weight:      &protoc.Laptop_WeightKg{WeightKg: randomFloat64(1.8, 3.6)},
		PriceUsd:    randomFloat64(500, 5000),
		ReleaseYear: uint32(2025),
		UpdatedAt:   timestamppb.Now(),
	}

	return laptop
}
