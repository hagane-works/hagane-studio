package imgcodec

import "testing"

func TestDecodeOwnDXT5DDS(t *testing.T) {
	w, h := 9, 7
	src := make([]byte, w*h*4)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := (y*w + x) * 4
			src[i], src[i+1], src[i+2], src[i+3] = byte(x*20), byte(y*30), 90, byte(100+x*10)
		}
	}
	dds := EncodeDXT5DDS(src, w, h)
	got, gw, gh, ok := DecodeDDS(dds)
	if !ok || gw != w || gh != h || len(got) != len(src) {
		t.Fatalf("ok=%v size=%dx%d bytes=%d", ok, gw, gh, len(got))
	}

	var color, alpha bool
	for i := 0; i < len(got); i += 4 {
		color = color || got[i] != 0 || got[i+1] != 0 || got[i+2] != 0
		alpha = alpha || got[i+3] != 0
	}
	if !color || !alpha {
		t.Fatal("decoded BC3 image is empty")
	}
}
