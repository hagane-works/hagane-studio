package mapgen

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"sort"
	"strings"

	"golang.org/x/image/bmp"
)

type Base struct {
	Width, Height int

	Province []int32

	Heights                      []uint8
	Terrain, Rivers              []uint8
	TerrainPalette, RiverPalette color.Palette

	Definitions []Definition
}

func LoadBase(provinces, heightmap, terrain, rivers, definitions []byte) (*Base, error) {
	decode := func(name string, data []byte) (image.Image, error) {
		img, err := bmp.Decode(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		return img, nil
	}
	provinceImg, err := decode("provinces.bmp", provinces)
	if err != nil {
		return nil, err
	}
	bounds := provinceImg.Bounds()
	b := &Base{Width: bounds.Dx(), Height: bounds.Dy(), Definitions: ParseDefinitions(definitions)}
	if len(b.Definitions) == 0 {
		return nil, errors.New("definition.csv: no province rows")
	}
	b.Province = provinceIDs(provinceImg, b.Definitions)

	indexed := func(name string, data []byte) (*image.Paletted, error) {
		img, err := decode(name, data)
		if err != nil {
			return nil, err
		}
		paletted, ok := img.(*image.Paletted)
		if !ok {
			return nil, fmt.Errorf("%s is not an 8-bit indexed image", name)
		}
		if paletted.Rect.Dx() != b.Width || paletted.Rect.Dy() != b.Height {
			return nil, fmt.Errorf("%s is %d×%d, provinces.bmp %d×%d", name, paletted.Rect.Dx(), paletted.Rect.Dy(), b.Width, b.Height)
		}
		return paletted, nil
	}
	heightImg, err := indexed("heightmap.bmp", heightmap)
	if err != nil {
		return nil, err
	}
	b.Heights = make([]uint8, len(b.Province))
	for p := range b.Heights {
		r, _, _, _ := heightImg.Palette[heightImg.Pix[(p/b.Width)*heightImg.Stride+p%b.Width]].RGBA()
		b.Heights[p] = uint8(r >> 8)
	}
	terrainImg, err := indexed("terrain.bmp", terrain)
	if err != nil {
		return nil, err
	}
	b.Terrain, b.TerrainPalette = packed(terrainImg), terrainImg.Palette
	riverImg, err := indexed("rivers.bmp", rivers)
	if err != nil {
		return nil, err
	}
	b.Rivers, b.RiverPalette = packed(riverImg), riverImg.Palette
	return b, nil
}

func LoadProvinces(provinces, definitions []byte) (width, height int, province []int32, err error) {
	img, err := bmp.Decode(bytes.NewReader(provinces))
	if err != nil {
		return 0, 0, nil, fmt.Errorf("provinces.bmp: %w", err)
	}
	rows := ParseDefinitions(definitions)
	if len(rows) == 0 {
		return 0, 0, nil, errors.New("definition.csv: no province rows")
	}
	return img.Bounds().Dx(), img.Bounds().Dy(), provinceIDs(img, rows), nil
}

func provinceIDs(img image.Image, definitions []Definition) []int32 {
	bounds := img.Bounds()
	width := bounds.Dx()
	idOf := make(map[uint32]int32, len(definitions))
	for _, d := range definitions {
		idOf[d.RGB] = int32(d.ID)
	}
	out := make([]int32, width*bounds.Dy())
	if rgba, ok := img.(*image.RGBA); ok {

		lastRGB, lastID := uint32(1<<31), int32(0)
		for p := range out {
			pix := rgba.Pix[(p/width)*rgba.Stride+(p%width)*4:]
			rgb := uint32(pix[0])<<16 | uint32(pix[1])<<8 | uint32(pix[2])
			if rgb != lastRGB {
				lastRGB, lastID = rgb, idOf[rgb]
			}
			out[p] = lastID
		}
		return out
	}
	for p := range out {
		r, g, bl, _ := img.At(bounds.Min.X+p%width, bounds.Min.Y+p/width).RGBA()
		out[p] = idOf[uint32(r>>8)<<16|uint32(g>>8)<<8|uint32(bl>>8)]
	}
	return out
}

func packed(img *image.Paletted) []uint8 {
	w, h := img.Rect.Dx(), img.Rect.Dy()
	if img.Stride == w {
		return append([]uint8(nil), img.Pix[:w*h]...)
	}
	out := make([]uint8, w*h)
	for y := 0; y < h; y++ {
		copy(out[y*w:(y+1)*w], img.Pix[y*img.Stride:])
	}
	return out
}

func (b *Base) definitionIndex() map[int]int {
	out := make(map[int]int, len(b.Definitions))
	for i, d := range b.Definitions {
		out[d.ID] = i
	}
	return out
}

func (b *Base) MaxProvinceID() int {
	most := 0
	for _, d := range b.Definitions {
		most = max(most, d.ID)
	}
	return most
}

func (b *Base) ProvincesImage() *image.RGBA {
	colorOf := make(map[int32]uint32, len(b.Definitions))
	for _, d := range b.Definitions {
		colorOf[int32(d.ID)] = d.RGB
	}
	out := image.NewRGBA(image.Rect(0, 0, b.Width, b.Height))
	for p, id := range b.Province {
		c := colorOf[id]
		out.Pix[p*4], out.Pix[p*4+1], out.Pix[p*4+2], out.Pix[p*4+3] = uint8(c>>16), uint8(c>>8), uint8(c), 255
	}
	return out
}

func (b *Base) HeightImage() *image.Paletted {
	out := image.NewPaletted(image.Rect(0, 0, b.Width, b.Height), GreyPalette())
	copy(out.Pix, b.Heights)
	return out
}

func (b *Base) TerrainImage() *image.Paletted {
	out := image.NewPaletted(image.Rect(0, 0, b.Width, b.Height), b.TerrainPalette)
	copy(out.Pix, b.Terrain)
	return out
}

func (b *Base) RiversImage() *image.Paletted {
	out := image.NewPaletted(image.Rect(0, 0, b.Width, b.Height), b.RiverPalette)
	copy(out.Pix, b.Rivers)
	return out
}

func (b *Base) DefinitionCSV() []byte { return DefinitionsCSV(b.Definitions) }

func DefinitionsCSV(definitions []Definition) []byte {
	rows := append([]Definition(nil), definitions...)
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	var out strings.Builder
	out.WriteString("0;0;0;0;land;false;unknown;0\n")
	for _, d := range rows {
		fmt.Fprintf(&out, "%d;%d;%d;%d;%s;%t;%s;%d\n", d.ID, d.RGB>>16, (d.RGB>>8)&0xff, d.RGB&0xff, d.Kind, d.Coastal, d.Terrain, d.Continent)
	}
	return []byte(out.String())
}

func (b *Base) CheckInput(terrainIndexes map[uint8]bool) CheckInput {
	colorOf := make(map[int32]uint32, len(b.Definitions))
	for _, d := range b.Definitions {
		colorOf[int32(d.ID)] = d.RGB
	}
	rgb := make([]uint32, len(b.Province))
	for p, id := range b.Province {
		rgb[p] = colorOf[id]
	}
	size := [2]int{b.Width, b.Height}
	in := CheckInput{
		Width: b.Width, Height: b.Height, ProvinceRGB: rgb, Definitions: b.Definitions,
		HeightSize: size, TerrainSize: size, RiverSize: size, Heights: b.Heights,
	}
	if terrainIndexes != nil {
		in.Terrain, in.TerrainIndexes = b.Terrain, terrainIndexes
	}
	return in
}
