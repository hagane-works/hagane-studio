package imgcodec

import "encoding/binary"

const ddpfLuminance = 0x20000

type channel struct {
	mask  uint32
	shift uint
	max   uint32
}

func newChannel(mask uint32) channel {
	if mask == 0 {
		return channel{}
	}
	shift := uint(0)
	for mask&(1<<shift) == 0 {
		shift++
	}
	return channel{mask: mask, shift: shift, max: mask >> shift}
}

func (c channel) value(raw uint32, fallback byte) byte {
	if c.mask == 0 || c.max == 0 {
		return fallback
	}
	v := (raw & c.mask) >> c.shift

	return byte((v*255 + c.max/2) / c.max)
}

func decodeUncompressedDDS(data []byte) (pix []byte, width, height int, ok bool) {
	if len(data) < ddsHeaderSize || string(data[:4]) != "DDS " {
		return nil, 0, 0, false
	}
	flags := binary.LittleEndian.Uint32(data[80:84])
	if flags&ddpfFourCC != 0 || flags&(ddpfRGB|ddpfLuminance) == 0 {
		return nil, 0, 0, false
	}
	bits := int(binary.LittleEndian.Uint32(data[88:92]))
	if bits != 16 && bits != 24 && bits != 32 {
		return nil, 0, 0, false
	}
	height = int(binary.LittleEndian.Uint32(data[12:16]))
	width = int(binary.LittleEndian.Uint32(data[16:20]))
	if width <= 0 || height <= 0 || width > maxDDSDimension || height > maxDDSDimension {
		return nil, 0, 0, false
	}
	red := newChannel(binary.LittleEndian.Uint32(data[92:96]))
	green := newChannel(binary.LittleEndian.Uint32(data[96:100]))
	blue := newChannel(binary.LittleEndian.Uint32(data[100:104]))
	alpha := newChannel(binary.LittleEndian.Uint32(data[104:108]))
	if flags&ddpfAlphaPixels == 0 {
		alpha = channel{}
	}
	if flags&ddpfLuminance != 0 && green.mask == 0 && blue.mask == 0 {

		green, blue = red, red
	}

	bytesPerPixel := bits / 8

	pitch := int(binary.LittleEndian.Uint32(data[20:24]))
	headerFlags := binary.LittleEndian.Uint32(data[8:12])
	if headerFlags&0x8 == 0 || pitch < width*bytesPerPixel {
		pitch = width * bytesPerPixel
	}
	body := data[ddsHeaderSize:]
	if len(body) < pitch*height {
		return nil, 0, 0, false
	}

	pix = make([]byte, width*height*4)
	for y := 0; y < height; y++ {
		row := body[y*pitch:]
		for x := 0; x < width; x++ {
			at := x * bytesPerPixel
			var raw uint32
			switch bytesPerPixel {
			case 2:
				raw = uint32(binary.LittleEndian.Uint16(row[at : at+2]))
			case 3:
				raw = uint32(row[at]) | uint32(row[at+1])<<8 | uint32(row[at+2])<<16
			default:
				raw = binary.LittleEndian.Uint32(row[at : at+4])
			}
			out := (y*width + x) * 4
			pix[out] = red.value(raw, 0)
			pix[out+1] = green.value(raw, 0)
			pix[out+2] = blue.value(raw, 0)
			pix[out+3] = alpha.value(raw, 255)
		}
	}
	return pix, width, height, true
}
