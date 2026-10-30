package guirender

import "testing"

func TestLabelsWithMarkupKeepIt(t *testing.T) {
	idx, err := NewIndex(map[string][]byte{
		"localisation/english/d_l_english.yml": []byte("l_english:\n" +
			" DESIGNER_ARMY_SAVE:0 \"Save £army_experience $VALUE|Y0$\"\n" +
			" PLAIN:0 \"Reset\"\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	save := idx.resolveText("DESIGNER_ARMY_SAVE")
	if save["markup"] != "Save £army_experience $VALUE|Y0$" {
		t.Errorf("markup = %v", save["markup"])
	}
	if _, ok := idx.resolveText("PLAIN")["markup"]; ok {
		t.Error("a plain label should carry no markup")
	}
}
