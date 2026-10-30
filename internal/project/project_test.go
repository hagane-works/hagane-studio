package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateWriteReopen(t *testing.T) {
	root := t.TempDir()
	store := NewStore(root)
	p, err := store.Create(Row{"name": "My Mod"}, "My Mod")
	if err != nil {
		t.Fatal(err)
	}
	if p.ID() != 1 || filepath.Base(p.Dir()) != "My Mod" {
		t.Fatalf("id %d dir %s", p.ID(), p.Dir())
	}
	err = p.Write([]string{"characters"}, func(tx *Tx) error {
		tx.Insert("characters", Row{"name": "A"})
		tx.Insert("characters", Row{"name": "B"})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	again, err := NewStore(root).Get(1)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := again.Rows("characters")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || Int64(rows[1]["id"]) != 2 || rows[1]["name"] != "B" {
		t.Fatalf("rows = %v", rows)
	}

	err = again.Write([]string{"characters"}, func(tx *Tx) error {
		tx.Delete("characters", func(r Row) bool { return Int64(r["id"]) == 2 })
		if row := tx.Insert("characters", Row{"name": "C"}); Int64(row["id"]) != 3 {
			t.Errorf("new id = %v, want 3", row["id"])
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestFailedWriteRestores(t *testing.T) {
	p, err := NewStore(t.TempDir()).Create(Row{"name": "x"}, "x")
	if err != nil {
		t.Fatal(err)
	}
	_ = p.Write([]string{"ideas"}, func(tx *Tx) error {
		tx.Insert("ideas", Row{"name": "kept"})
		return nil
	})
	err = p.Write([]string{"ideas"}, func(tx *Tx) error {
		tx.Insert("ideas", Row{"name": "dropped"})
		return os.ErrInvalid
	})
	if err != os.ErrInvalid {
		t.Fatalf("err = %v", err)
	}
	rows, _ := p.Rows("ideas")
	if len(rows) != 1 || rows[0]["name"] != "kept" {
		t.Fatalf("rows after failed write = %v", rows)
	}
}

func TestBaseModsBesideTheRecord(t *testing.T) {
	root := t.TempDir()
	p, err := NewStore(root).Create(Row{"name": "Submod"}, "Submod")
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.UpdateRecordAndBaseMods(func(record Row, baseMods map[string]string) {
		record["dependencies"] = []any{"Base"}
		baseMods["Base"] = "D:/mods/base"
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := p.UpdateRecord(func(record Row) { record["name"] = "Renamed" }); err != nil {
		t.Fatal(err)
	}
	again, err := NewStore(root).Get(p.ID())
	if err != nil {
		t.Fatal(err)
	}
	if got := again.BaseMods(); len(got) != 1 || got["Base"] != "D:/mods/base" {
		t.Fatalf("base mods = %v", got)
	}
	if record := again.Record(); record["name"] != "Renamed" || record["base_mods"] != nil {
		t.Fatalf("record = %v", record)
	}
}

func TestCopiedFolderGetsNewID(t *testing.T) {
	root := t.TempDir()
	store := NewStore(root)
	p, err := store.Create(Row{"name": "a"}, "a")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(p.Dir(), metaFile))
	if err != nil {
		t.Fatal(err)
	}
	copyDir := filepath.Join(root, "b")
	if err := os.Mkdir(copyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(copyDir, metaFile), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	rows, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	ids := map[int64]bool{}
	for _, row := range rows {
		ids[Int64(row["id"])] = true
	}
	if len(rows) != 2 || len(ids) != 2 {
		t.Fatalf("rows = %v", rows)
	}
}

func TestFolderName(t *testing.T) {
	for in, want := range map[string]string{
		"My Mod":         "My Mod",
		"中华民国 1936":      "中华民国 1936",
		"a/b:c":          "a_b_c",
		"  ":             "project",
		"Kaiserreich...": "Kaiserreich",
	} {
		if got := FolderName(in); got != want {
			t.Errorf("FolderName(%q) = %q, want %q", in, got, want)
		}
	}
}
