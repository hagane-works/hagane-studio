package guirender

import "testing"

func TestConsumerGoodsFactories(t *testing.T) {
	idx, err := NewIndex(map[string][]byte{
		"common/defines/00_defines.lua": []byte("NDefines.NProduction.MINIMUM_NUMBER_OF_FACTORIES_TAKEN_BY_CONSUMER_GOODS_VALUE = 1\n" +
			"NDefines.NProduction.MINIMUM_NUMBER_OF_FACTORIES_TAKEN_BY_CONSUMER_GOODS_PERCENT = 0.1\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		factories        int
		expected, factor float64
		want             int
	}{
		{63, 0.25, 0, 15},
		{63, 0.25, 0.1, 17},
		{40, 0.05, 0, 4},
		{3, 0, 0, 1},
		{0, 0.3, 0, 0},
	}
	for _, c := range cases {
		if got := idx.consumerGoodsFactories(c.factories, c.expected, c.factor); got != c.want {
			t.Errorf("%d × %v × (1+%v) = %d, want %d", c.factories, c.expected, c.factor, got, c.want)
		}
	}
}
