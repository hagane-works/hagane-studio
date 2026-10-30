package locout

import (
	"reflect"
	"testing"
)

func TestEnsureEnFirstStringsMatchesPython(t *testing.T) {
	cases := []struct {
		in, want []string
	}{
		{nil, []string{"en"}},
		{[]string{}, []string{"en"}},
		{[]string{" zh ", "en", "RU", "zh", "xx"}, []string{"en", "zh", "ru"}},
	}
	for _, tc := range cases {
		if got := EnsureEnFirstStrings(tc.in); !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("EnsureEnFirstStrings(%v)=%v want %v", tc.in, got, tc.want)
		}
	}
}
