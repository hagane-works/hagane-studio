package imgcodec

import "encoding/binary"

func DecodeDDS(data []byte) (pix []byte, width, height int, ok bool) {
	if pix, width, height, ok = FastDecodeDDS(data); ok {
		return pix, width, height, true
	}
	if len(data) < ddsHeaderSize || string(data[:4]) != "DDS " || binary.LittleEndian.Uint32(data[4:8]) != 124 {
		return nil, 0, 0, false
	}
	height = int(binary.LittleEndian.Uint32(data[12:16]))
	width = int(binary.LittleEndian.Uint32(data[16:20]))
	if width <= 0 || height <= 0 || width > maxDDSDimension || height > maxDDSDimension {
		return nil, 0, 0, false
	}
	fourCC := string(data[84:88])
	blockBytes := 16
	if fourCC == "DXT1" {
		blockBytes = 8
	} else if fourCC != "DXT3" && fourCC != "DXT5" {

		return decodeUncompressedDDS(data)
	}
	bw, bh := (width+3)/4, (height+3)/4
	need := bw * bh * blockBytes
	if len(data)-ddsHeaderSize < need {
		return nil, 0, 0, false
	}
	pix = make([]byte, width*height*4)
	src := data[ddsHeaderSize : ddsHeaderSize+need]
	for by := 0; by < bh; by++ {
		for bx := 0; bx < bw; bx++ {
			block := src[(by*bw+bx)*blockBytes:]
			var rgba [16][4]byte
			switch fourCC {
			case "DXT1":
				decodeBC1(block[:8], &rgba, true)
			case "DXT3":
				decodeBC2(block[:16], &rgba)
			case "DXT5":
				decodeBC3(block[:16], &rgba)
			}
			writeBlock(pix, width, height, bx, by, &rgba)
		}
	}
	return pix, width, height, true
}

func rgb565(value uint16) [4]byte {
	r := byte((value >> 11) & 31)
	g := byte((value >> 5) & 63)
	b := byte(value & 31)
	return [4]byte{(r << 3) | (r >> 2), (g << 2) | (g >> 4), (b << 3) | (b >> 2), 255}
}

func colorTable(block []byte, transparent bool) [4][4]byte {
	c0v, c1v := binary.LittleEndian.Uint16(block[:2]), binary.LittleEndian.Uint16(block[2:4])
	c0, c1 := rgb565(c0v), rgb565(c1v)
	table := [4][4]byte{c0, c1}
	if transparent && c0v <= c1v {
		for i := 0; i < 3; i++ {
			table[2][i] = byte((int(c0[i]) + int(c1[i])) / 2)
		}
		table[2][3] = 255
		table[3] = [4]byte{}
	} else {
		for i := 0; i < 3; i++ {
			table[2][i] = byte((2*int(c0[i]) + int(c1[i])) / 3)
			table[3][i] = byte((int(c0[i]) + 2*int(c1[i])) / 3)
		}
		table[2][3], table[3][3] = 255, 255
	}
	return table
}

func decodeBC1(block []byte, out *[16][4]byte, transparent bool) {
	table := colorTable(block, transparent)
	indices := binary.LittleEndian.Uint32(block[4:8])
	for i := 0; i < 16; i++ {
		out[i] = table[(indices>>uint(2*i))&3]
	}
}

func decodeBC2(block []byte, out *[16][4]byte) {
	decodeBC1(block[8:16], out, false)
	alpha := binary.LittleEndian.Uint64(block[:8])
	for i := 0; i < 16; i++ {
		a := byte((alpha >> uint(4*i)) & 15)
		out[i][3] = a | a<<4
	}
}

func decodeBC3(block []byte, out *[16][4]byte) {
	decodeBC1(block[8:16], out, false)
	a0, a1 := block[0], block[1]
	var table [8]byte
	table[0], table[1] = a0, a1
	if a0 > a1 {
		for i := 1; i <= 6; i++ {
			table[i+1] = byte(((7-i)*int(a0) + i*int(a1)) / 7)
		}
	} else {
		for i := 1; i <= 4; i++ {
			table[i+1] = byte(((5-i)*int(a0) + i*int(a1)) / 5)
		}
		table[6], table[7] = 0, 255
	}
	var bits uint64
	for i := 0; i < 6; i++ {
		bits |= uint64(block[2+i]) << uint(8*i)
	}
	for i := 0; i < 16; i++ {
		out[i][3] = table[(bits>>uint(3*i))&7]
	}
}

func writeBlock(dst []byte, width, height, bx, by int, block *[16][4]byte) {
	for y := 0; y < 4; y++ {
		py := by*4 + y
		if py >= height {
			continue
		}
		for x := 0; x < 4; x++ {
			px := bx*4 + x
			if px >= width {
				continue
			}
			copy(dst[(py*width+px)*4:], block[y*4+x][:])
		}
	}
}
