package mapgen

import "math"

func boxBlur(field []float32, width, height, radius int) []float32 {
	if radius < 1 {
		return field
	}
	tmp := make([]float32, len(field))
	out := make([]float32, len(field))
	for y := 0; y < height; y++ {
		row := field[y*width : (y+1)*width]
		var sum float32
		n := 0
		for x := -radius; x <= radius; x++ {
			if x >= 0 && x < width {
				sum += row[x]
				n++
			}
		}
		for x := 0; x < width; x++ {
			tmp[y*width+x] = sum / float32(n)
			if out := x - radius; out >= 0 {
				sum -= row[out]
				n--
			}
			if in := x + radius + 1; in < width {
				sum += row[in]
				n++
			}
		}
	}
	for x := 0; x < width; x++ {
		var sum float32
		n := 0
		for y := -radius; y <= radius; y++ {
			if y >= 0 && y < height {
				sum += tmp[y*width+x]
				n++
			}
		}
		for y := 0; y < height; y++ {
			out[y*width+x] = sum / float32(n)
			if gone := y - radius; gone >= 0 {
				sum -= tmp[gone*width+x]
				n--
			}
			if in := y + radius + 1; in < height {
				sum += tmp[in*width+x]
				n++
			}
		}
	}
	return out
}

func heights(classes []Class, width, height int, seed int64) []uint8 {
	base := make([]float32, len(classes))
	rough := make([]float32, len(classes))
	for i, class := range classes {
		entry := entryOf(class)
		base[i] = float32(entry.Height)
		rough[i] = float32(entry.Rough)
	}
	radius := int(math.Max(2, float64(width)/450))
	base = boxBlur(base, width, height, radius)
	rough = boxBlur(rough, width, height, radius)
	cell := math.Max(8, float64(width)/90)
	out := make([]uint8, len(classes))
	for i, class := range classes {
		if !class.Land() {
			out[i] = SeaHeight
			continue
		}
		x, y := float64(i%width), float64(i/width)
		v := float64(base[i]) + float64(rough[i])*fractalNoise(seed+17, x, y, cell) + 1.5*valueNoise(seed+29, x, y, 4)
		out[i] = uint8(math.Max(LandHeight, math.Min(255, math.Round(v))))
	}
	return out
}
