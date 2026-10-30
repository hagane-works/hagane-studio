package mapgen

import "math"

type Class uint8

const (
	Sea Class = iota
	Plains
	Forest
	Hills
	Mountain
	Desert
	Marsh
	Jungle

	Lake
)

func (c Class) Land() bool { return c != Sea && c != Lake }

type Entry struct {
	Class Class

	Key string

	Name string

	Color [3]uint8

	Terrain uint8

	Height float64
	Rough  float64
}

var Legend = []Entry{
	{Sea, "ocean", "海", [3]uint8{40, 80, 170}, 15, 89, 0},
	{Plains, "plains", "平原", [3]uint8{160, 210, 100}, 0, 101, 2},
	{Forest, "forest", "森林", [3]uint8{30, 110, 40}, 1, 104, 3},
	{Hills, "hills", "丘陵", [3]uint8{200, 160, 90}, 17, 124, 8},
	{Mountain, "mountain", "山地", [3]uint8{120, 100, 90}, 6, 165, 24},
	{Desert, "desert", "沙漠", [3]uint8{245, 225, 140}, 3, 101, 3},
	{Marsh, "marsh", "沼泽", [3]uint8{100, 150, 140}, 9, 97, 1},
	{Jungle, "jungle", "丛林", [3]uint8{0, 170, 110}, 21, 103, 3},
	{Lake, "lakes", "湖", [3]uint8{120, 180, 235}, 14, 89, 0},
}

func entryOf(c Class) Entry { return Legend[c] }

const (
	SeaHeight  = 89
	LandHeight = 95
)

const (
	RiverLand  = 255
	RiverWater = 254
)

func nearestClass(r, g, b uint8) Class {
	best, bestDistance := Sea, math.MaxFloat64
	for _, entry := range Legend {
		dr := float64(r) - float64(entry.Color[0])
		dg := float64(g) - float64(entry.Color[1])
		db := float64(b) - float64(entry.Color[2])

		distance := 2*dr*dr + 4*dg*dg + 3*db*db
		if distance < bestDistance {
			best, bestDistance = entry.Class, distance
		}
	}
	return best
}
