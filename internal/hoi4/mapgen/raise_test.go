package mapgen

import "testing"

func TestRaiseWaterProvince(t *testing.T) {
	b := &Base{
		Width: 4, Height: 1,
		Province: []int32{1, 1, 2, 2},
		Heights:  []uint8{100, 100, SeaHeight, SeaHeight},
		Terrain:  []uint8{0, 0, Legend[Sea].Terrain, Legend[Sea].Terrain},
		Rivers:   []uint8{RiverLand, RiverLand, RiverWater, RiverWater},
		Definitions: []Definition{
			{ID: 1, Kind: "land", Terrain: "plains", Continent: 1},
			{ID: 2, Kind: "sea", Terrain: "ocean"},
		},
	}
	add, err := b.Raise(2, "hills", 1)
	if err != nil {
		t.Fatal(err)
	}
	if add.Changed.Empty() || len(add.Provinces) != 1 {
		t.Fatalf("addition = %+v", add)
	}
	if row := b.Definitions[1]; row.Kind != "land" || row.Terrain != "hills" || row.Continent != 1 || row.Coastal {
		t.Errorf("definition = %+v", row)
	}
	if b.Heights[2] < LandHeight || b.Terrain[2] != Legend[Hills].Terrain || b.Rivers[2] != RiverLand {
		t.Errorf("pixels = height %d terrain %d river %d", b.Heights[2], b.Terrain[2], b.Rivers[2])
	}
}
