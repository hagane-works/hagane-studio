package guirender

import "testing"

func TestSourceArchiveRevisionSurvivesLayeringAndFileMasks(t *testing.T) {
	imported, err := NewIndex(map[string][]byte{
		"interface/imported.gui": []byte(`guiTypes = { }`),
	})
	if err != nil {
		t.Fatal(err)
	}
	imported.SetSourceArchiveRevision("snapshot.zip", "revision-1")
	masked, err := imported.WithoutFiles([]string{"interface/imported.gui"})
	if err != nil {
		t.Fatal(err)
	}
	project, err := NewIndexWithFallback(map[string][]byte{}, masked)
	if err != nil {
		t.Fatal(err)
	}
	key, revision, ok := project.SourceArchiveRevision()
	if !ok || key != "snapshot.zip" || revision != "revision-1" {
		t.Fatalf("source archive = %q %q %v", key, revision, ok)
	}
}
