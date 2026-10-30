package importjob

import "testing"

func TestDecodeUTF16AndNUL(t *testing.T) {
	got := decodeScriptBytes([]byte{0xff, 0xfe, 'x', 0, '=', 0, '1', 0})
	if got != "x=1" {
		t.Fatalf("got %q", got)
	}
}
