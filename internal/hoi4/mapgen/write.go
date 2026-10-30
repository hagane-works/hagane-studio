package mapgen

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"strings"

	"golang.org/x/image/bmp"
)

func (m *Map) ProvincesImage() *image.RGBA {
	out := image.NewRGBA(image.Rect(0, 0, m.Width, m.Height))
	for p, k := range m.Province {
		c := m.Provinces[k].Color
		out.Pix[p*4], out.Pix[p*4+1], out.Pix[p*4+2], out.Pix[p*4+3] = c[0], c[1], c[2], 255
	}
	return out
}

func (m *Map) HeightImage() *image.Gray {
	out := image.NewGray(image.Rect(0, 0, m.Width, m.Height))
	copy(out.Pix, m.Heights)
	return out
}

func (m *Map) TerrainImage(palette color.Palette) *image.Paletted {
	out := image.NewPaletted(image.Rect(0, 0, m.Width, m.Height), palette)
	for p, class := range m.Classes {
		out.Pix[p] = entryOf(class).Terrain
	}
	return out
}

func (m *Map) RiversImage(palette color.Palette) *image.Paletted {
	out := image.NewPaletted(image.Rect(0, 0, m.Width, m.Height), palette)
	for p, class := range m.Classes {
		if class.Land() {
			out.Pix[p] = RiverLand
		} else {
			out.Pix[p] = RiverWater
		}
	}
	return out
}

func (m *Map) DefinitionCSV() []byte {
	var b strings.Builder
	b.WriteString("0;0;0;0;land;false;unknown;0\n")
	for _, p := range m.Provinces {
		fmt.Fprintf(&b, "%d;%d;%d;%d;%s;%t;%s;%d\n", p.ID, p.Color[0], p.Color[1], p.Color[2], p.Kind, p.Coastal, p.Terrain, p.Continent)
	}
	return []byte(b.String())
}

func EncodeBMP(img image.Image) ([]byte, error) {
	var out bytes.Buffer
	if err := bmp.Encode(&out, img); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func GreyPalette() color.Palette {
	out := make(color.Palette, 256)
	for i := range out {
		out[i] = color.RGBA{uint8(i), uint8(i), uint8(i), 255}
	}
	return out
}
