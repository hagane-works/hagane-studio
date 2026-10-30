package ideas

import (
	"math/big"
	"testing"

	"haganeworks/internal/hoi4/importsvc"
)

func TestNativeEntity(t *testing.T) {
	d := NewDict()
	d.Set("id", big.NewInt(7))
	d.Set("scope", importsvc.ScopeVerdict{Kind: importsvc.ScopeOwn, CountryTags: []string{"USA"}})
	got := Native(d).(map[string]any)
	if got["id"] != int64(7) || got["scope"].(map[string]any)["country_tag"] != "USA" {
		t.Fatalf("unexpected native entity: %#v", got)
	}
}
