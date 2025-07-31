package sample

import (
	"math/rand/v2"

	"github.com/go-http-server/grpc/protoc"
)

func randomInt(min, max int) int {
	return min + rand.IntN(max-min+1)
}

func randomBool() bool {
	return rand.IntN(2) == 1
}

func randomFloat64(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}

func randomFloat32(min, max float32) float32 {
	return min + rand.Float32()*(max-min)
}

func randomGPUBrand() string {
	return randomStringFromSet("NVIDIA", "AMD", "Intel")
}

func randomKeyboardLayout() protoc.Keyboard_Layout {
	layouts := []protoc.Keyboard_Layout{
		protoc.Keyboard_QWERTY,
		protoc.Keyboard_QWERTZ,
		protoc.Keyboard_AZERTY,
	}
	return layouts[randomInt(0, len(layouts)-1)]
}

func randomStringFromSet(values ...string) string {
	n := len(values)
	if n == 0 {
		return ""
	}

	return values[rand.IntN(n)]
}

func randomCPUBrand() string {
	return randomStringFromSet("Intel", "AMD")
}

func randomCPUName(brand string) string {
	switch brand {
	case "Intel":
		{
			return randomStringFromSet(
				"Core i3-13100",
				"Core i5-12400F",
				"Core i5-13500",
				"Core i7-12700K",
				"Core i7-13700H",
				"Core i9-13900K",
				"Core i9-14900HX",
				"Core Ultra 7 155H",
				"Xeon W-2445",
			)
		}
	case "AMD":
		{
			return randomStringFromSet(
				"Ryzen 5 5600X",
				"Ryzen 5 7600",
				"Ryzen 7 5800X3D",
				"Ryzen 7 7700X",
				"Ryzen 9 5900X",
				"Ryzen 9 7900X3D",
				"Ryzen Threadripper 7980X",
				"EPYC 9654",
				"Athlon Gold 3150G",
			)
		}
	default:
		return ""
	}
}

func randomGPUName(brand string) string {
	switch brand {
	case "NVIDIA":
		return randomStringFromSet(
			"GeForce GTX 1660 Super",
			"GeForce RTX 3060",
			"GeForce RTX 3070 Ti",
			"GeForce RTX 4080",
			"GeForce RTX 4090",
			"Quadro RTX A6000",
			"Tesla V100",
			"A100",
			"H100",
			"B200",
		)
	case "AMD":
		return randomStringFromSet(
			"Radeon RX 6600 XT",
			"Radeon RX 6700 XT",
			"Radeon RX 6800",
			"Radeon RX 7900 XTX",
			"Radeon PRO W7900",
			"Instinct MI300X",
		)
	case "Intel":
		return randomStringFromSet(
			"Intel Iris Xe",
			"Intel Arc A770",
			"Intel Arc A750",
		)
	default:
		return ""
	}
}

func randomScreenPanel() protoc.Screen_Panel {
	panels := []protoc.Screen_Panel{
		protoc.Screen_IPS,
		protoc.Screen_OLED,
	}

	return panels[randomInt(0, len(panels)-1)]
}

func NewMemory() *protoc.Memory {
	return &protoc.Memory{
		Value: uint64(randomInt(4, 128)),
		Unit:  protoc.Memory_GIGABYTE,
	}
}

func randomScreenResolution() *protoc.Screen_Resolution {
	height := randomInt(1080, 4320) // Random height between 1080 and 2160
	width := height * 16 / 9        // Assuming a 16:9 aspect ratio

	return &protoc.Screen_Resolution{
		Width:  uint32(width),
		Height: uint32(height),
	}
}

func randomLaptopBrand() string {
	return randomStringFromSet(
		"Dell",
		"HP",
		"Lenovo",
		"Acer",
		"Asus",
	)
}

func randomLaptopName(brand string) string {
	switch brand {
	case "Dell":
		return randomStringFromSet(
			"XPS 13",
			"XPS 15",
			"Inspiron 15",
			"Latitude 7430",
			"Alienware x16",
		)
	case "HP":
		return randomStringFromSet(
			"Spectre x360",
			"Envy 13",
			"Pavilion 15",
			"EliteBook 840 G10",
			"OMEN 16",
		)
	case "Lenovo":
		return randomStringFromSet(
			"ThinkPad X1 Carbon",
			"ThinkPad T14",
			"IdeaPad Slim 5",
			"Yoga 9i",
			"Legion 5 Pro",
		)
	case "Acer":
		return randomStringFromSet(
			"Aspire 5",
			"Swift X",
			"Spin 3",
			"Nitro 5",
			"Predator Helios 300",
		)
	case "Asus":
		return randomStringFromSet(
			"ZenBook 14",
			"Vivobook 15",
			"TUF Gaming F15",
			"ROG Zephyrus G14",
			"ExpertBook B9",
		)
	default:
		return ""
	}
}
