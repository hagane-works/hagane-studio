package imgcodec

import "encoding/binary"

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func encode565(r, g, b uint8) uint16 {
	return uint16(r>>3)<<11 | uint16(g>>2)<<5 | uint16(b>>3)
}

func decode565(x uint16) [3]uint8 {
	r := int(x&0xf800) >> 8
	r |= r >> 5
	g := int(x&0x7e0) >> 3
	g |= g >> 6
	b := int(x&0x1f) << 3
	b |= b >> 5
	return [3]uint8{uint8(r), uint8(g), uint8(b)}
}

func encodeBC3Alpha(pix []byte, stride, w, h, bx, by int, dst []byte) {
	var block [16]uint8
	var alphaMin, alphaMax uint8
	first := true
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			x, y := bx+i, by+j
			if x >= w || y >= h {
				block[i+j*4] = 0
				continue
			}
			a := pix[y*stride+x*4+3]
			block[i+j*4] = a
			if first || a < alphaMin {
				alphaMin = a
			}
			if first || a > alphaMax {
				alphaMax = a
			}
			first = false
		}
	}
	dst[0] = alphaMin
	dst[1] = alphaMax

	denom := float32(absInt(int(alphaMax) - int(alphaMin)))
	for i := 0; i < 2; i++ {
		var l uint32
		for j := 7; j > -1; j-- {
			a := block[i*8+j]
			if a == 0 {
				l |= 6 << (j * 3)
				continue
			} else if a == 255 {
				l |= 7 << (j * 3)
				continue
			}
			var distance float32
			if denom != 0 {
				distance = float32(absInt(int(a)-int(alphaMin))) / denom * 10
			}
			switch {
			case distance < 3:
				l |= 2 << (j * 3)
			case distance < 5:
				l |= 3 << (j * 3)
			case distance < 7:
				l |= 4 << (j * 3)
			default:
				l |= 5 << (j * 3)
			}
		}
		dst[2+i*3] = uint8(l)
		dst[3+i*3] = uint8(l >> 8)
		dst[4+i*3] = uint8(l >> 16)
	}
}

func encodeBC1Color(pix []byte, stride, w, h, bx, by int, dst []byte) {
	var block [16][3]uint8
	var colorMin, colorMax uint16
	first := true
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			x, y := bx+i, by+j
			idx := i + j*4
			if x >= w || y >= h {

				block[idx] = [3]uint8{0, 0, 0}
				continue
			}
			o := y*stride + x*4
			block[idx][0] = pix[o]
			block[idx][1] = pix[o+1]
			block[idx][2] = pix[o+2]
			c := encode565(block[idx][0], block[idx][1], block[idx][2])
			if first || c < colorMin {
				colorMin = c
			}
			if first || c > colorMax {
				colorMax = c
			}
			first = false
		}
	}

	binary.LittleEndian.PutUint16(dst[0:2], colorMax)
	binary.LittleEndian.PutUint16(dst[2:4], colorMin)

	minRGB := decode565(colorMin)
	maxRGB := decode565(colorMax)
	for i := 0; i < 4; i++ {
		var l uint8
		for j := 3; j > -1; j-- {
			cur := block[i*4+j]
			var distance float32
			total := 0
			for k := 0; k < 3; k++ {
				denom := float32(absInt(int(maxRGB[k]) - int(minRGB[k])))
				if denom != 0 {
					distance += float32(absInt(int(cur[k])-int(minRGB[k]))) / denom
					total++
				}
			}
			if total == 0 {
				continue
			}

			distance *= float32(6 / total)
			switch {
			case distance < 1:
				l |= 1 << (j * 2)
			case distance < 3:
				l |= 3 << (j * 2)
			case distance < 5:
				l |= 2 << (j * 2)
			default:

			}
		}
		dst[4+i] = l
	}
}

func EncodeDXT5Blocks(pix []byte, w, h int) []byte {
	if w <= 0 || h <= 0 || len(pix) != w*h*4 {
		return nil
	}
	stride := w * 4
	bw, bh := (w+3)/4, (h+3)/4
	out := make([]byte, bw*bh*16)
	p := 0
	for by := 0; by < h; by += 4 {
		for bx := 0; bx < w; bx += 4 {
			encodeBC3Alpha(pix, stride, w, h, bx, by, out[p:p+8])
			encodeBC1Color(pix, stride, w, h, bx, by, out[p+8:p+16])
			p += 16
		}
	}
	return out
}
