package world

import (
	"math/big"
	"testing"
)

const forceLinkStateText = `state = {
	id = 298
	name = "STATE_298"
	impassable = yes
	state_category = wasteland
	force_link_ownership_to = 334
	manpower = 500
	history = {
		owner = INS
	}
	provinces = { 12654 }
}
`

func TestImportStateForceLinkOwnershipTo_ColumnAndRawExtrasCopy(t *testing.T) {
	rows, summary := ImportStateFiles(forceLinkStateText, "history/states/298-Interior Borneo.txt")
	if len(rows) != 1 {
		t.Fatalf("expected 1 state, got %d", len(rows))
	}
	row := rows[0]

	got, _ := row.Get("force_link_ownership_to")
	z, ok := got.(*big.Int)
	if !ok || z.Int64() != 334 {
		t.Fatalf("force_link_ownership_to column = %#v, want big.Int 334", got)
	}

	rawExtras, _ := row.Get("raw_extras")
	rx, ok := rawExtras.(*PyDict)
	if !ok {
		t.Fatalf("raw_extras is %T, want *PyDict", rawExtras)
	}
	copyVal, present := rx.Get("force_link_ownership_to")
	if !present {
		t.Fatalf("raw_extras 副本不见了 —— 两条发射路径读的都是它,"+
			"列里有值但导出会静默丢掉这个字段。raw_extras keys=%v", rx.Keys())
	}
	if s, _ := copyVal.(string); s != "334" {
		t.Fatalf("raw_extras copy = %#v, want %q (importsvc.ScalarToText 形状)", copyVal, "334")
	}

	found := false
	for _, p := range summary.PreservedFields.Keys {
		if p == "state.force_link_ownership_to" {
			found = true
		}
	}
	if !found {
		t.Fatalf("summary.PreservedFields 缺 state.force_link_ownership_to,got %v",
			summary.PreservedFields.Keys)
	}
}

func TestImportStateForceLinkOwnershipTo_NonIntegerKeepsCopyOnly(t *testing.T) {
	text := `state = {
	id = 299
	force_link_ownership_to = SOMETHING
}
`
	rows, _ := ImportStateFiles(text, "history/states/299-X.txt")
	if len(rows) != 1 {
		t.Fatalf("expected 1 state, got %d", len(rows))
	}
	if v, _ := rows[0].Get("force_link_ownership_to"); v != nil {
		t.Fatalf("column should stay nil for a non-integer value, got %#v", v)
	}
	rx, _ := rows[0].Get("raw_extras")
	copyVal, present := rx.(*PyDict).Get("force_link_ownership_to")
	if !present || copyVal.(string) != "SOMETHING" {
		t.Fatalf("raw_extras copy = %#v (present=%v), want %q", copyVal, present, "SOMETHING")
	}
}

func TestImportStateForceLinkOwnershipTo_AbsentStaysAbsent(t *testing.T) {
	text := `state = {
	id = 300
	name = "STATE_300"
	provinces = { 1 }
}
`
	rows, _ := ImportStateFiles(text, "history/states/300-Y.txt")
	if len(rows) != 1 {
		t.Fatalf("expected 1 state, got %d", len(rows))
	}
	if v, _ := rows[0].Get("force_link_ownership_to"); v != nil {
		t.Fatalf("column should be nil, got %#v", v)
	}
	rx, _ := rows[0].Get("raw_extras")
	if _, present := rx.(*PyDict).Get("force_link_ownership_to"); present {
		t.Fatalf("raw_extras 不该凭空多出 force_link_ownership_to")
	}
}
