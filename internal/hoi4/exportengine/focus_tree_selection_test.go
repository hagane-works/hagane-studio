package exportengine

import (
	"encoding/json"
	"strings"
	"testing"
)

func focusSnapshot(tree string, nodes ...string) Snapshot {
	nodeRaws := make([]json.RawMessage, 0, len(nodes))
	for _, n := range nodes {
		nodeRaws = append(nodeRaws, json.RawMessage(n))
	}
	return Snapshot{
		ProjectID: 42,
		Project:   []byte(`{"id":42,"name":"Demo","country_tag":"ROM","export_languages":["english"]}`),
		Tables: map[string][]json.RawMessage{
			"focus_trees": {json.RawMessage(tree)},
			"focus_nodes": nodeRaws,
		},
	}
}

func warningsContaining(artifact NativeArtifact, needle string) []string {
	var out []string
	for _, w := range artifact.Warnings {
		if strings.Contains(w, needle) {
			out = append(out, w)
		}
	}
	return out
}

const oneNode = `{"id":1,"focus_tree_id":7,"focus_id":"rom_a","x":1,"y":1,"name_en":"A"}`

func TestFocusTreeWithoutCountryFallsBackToProjectPrimary(t *testing.T) {
	artifact, err := BuildNativeShadow(focusSnapshot(
		`{"id":7,"tree_id":"rom_tree","country_tag":"","scope_kind":"own","source":"created","tree_kind":"primary"}`,
		oneNode), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	_, script := focusScript(t, artifact)
	if !strings.Contains(script, "tag = ROM") {
		t.Fatalf("tree must become selectable by the project's primary country:\n%s", script)
	}
	if !strings.Contains(script, "country = {") {
		t.Fatalf("no country block at all — nothing can select this tree:\n%s", script)
	}
	hits := warningsContaining(artifact, "rom_tree")
	if len(hits) != 1 {
		t.Fatalf("the author must be told the tree was assigned for them, got %v", artifact.Warnings)
	}
	if !strings.Contains(hits[0], "ROM") {
		t.Fatalf("the notice must name the country it picked: %s", hits[0])
	}
}

func TestFocusTreeWithoutAnyCountryWarnsInsteadOfGuessing(t *testing.T) {
	snapshot := focusSnapshot(
		`{"id":7,"tree_id":"orphan_tree","country_tag":"","scope_kind":"own","source":"created","tree_kind":"primary"}`,
		oneNode)
	snapshot.Project = []byte(`{"id":42,"name":"Demo","country_tag":"","export_languages":["english"]}`)
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if hits := warningsContaining(artifact, "orphan_tree"); len(hits) != 1 {
		t.Fatalf("an unselectable tree must be reported, got %v", artifact.Warnings)
	}
	if _, body := focusScript(t, artifact); strings.Contains(body, "country = {") {
		t.Fatalf("no country is known — inventing a country block would be a lie:\n%s", body)
	}
}

func TestGlobalScopeEmitsAllCountriesBlock(t *testing.T) {
	artifact, err := BuildNativeShadow(focusSnapshot(
		`{"id":7,"tree_id":"global_tree","country_tag":"","scope_kind":"global","source":"created","tree_kind":"primary"}`,
		oneNode), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	_, script := focusScript(t, artifact)
	if !strings.Contains(script, "country = {") || !strings.Contains(script, "factor = 100") {
		t.Fatalf("a global tree must score for every country:\n%s", script)
	}
	if hits := warningsContaining(artifact, "global_tree"); len(hits) != 0 {
		t.Fatalf("a handled scope must not warn: %v", hits)
	}
}

func TestSharedPoolScopeStaysSilent(t *testing.T) {
	artifact, err := BuildNativeShadow(focusSnapshot(
		`{"id":7,"tree_id":"pool_tree","country_tag":"","scope_kind":"shared_pool","source":"created","tree_kind":"primary"}`,
		oneNode), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if hits := warningsContaining(artifact, "pool_tree"); len(hits) != 0 {
		t.Fatalf("a shared pool legitimately has no country block: %v", hits)
	}
}

func TestUnknownScopeIsReported(t *testing.T) {
	artifact, err := BuildNativeShadow(focusSnapshot(
		`{"id":7,"tree_id":"weird_tree","country_tag":"","scope_kind":"someday_maybe","source":"created","tree_kind":"primary"}`,
		oneNode), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if hits := warningsContaining(artifact, "weird_tree"); len(hits) != 1 {
		t.Fatalf("an unrecognised scope must be reported, got %v", artifact.Warnings)
	}
}

func TestEmptyUnselectableTreeStaysSilent(t *testing.T) {
	artifact, err := BuildNativeShadow(focusSnapshot(
		`{"id":7,"tree_id":"empty_tree","country_tag":"","scope_kind":"someday_maybe","source":"created","tree_kind":"primary"}`,
	), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if hits := warningsContaining(artifact, "empty_tree"); len(hits) != 0 {
		t.Fatalf("a tree with no focuses must not warn: %v", hits)
	}
}

func TestTreeWithItsOwnCountryIsUnchanged(t *testing.T) {
	artifact, err := BuildNativeShadow(focusSnapshot(
		`{"id":7,"tree_id":"rom_tree","country_tag":"ROM","scope_kind":"own","source":"created","tree_kind":"primary"}`,
		oneNode), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	_, script := focusScript(t, artifact)
	if !strings.Contains(script, "add = 100") || !strings.Contains(script, "tag = ROM") {
		t.Fatalf("the ordinary path must keep its existing shape:\n%s", script)
	}
	if hits := warningsContaining(artifact, "rom_tree"); len(hits) != 0 {
		t.Fatalf("nothing is wrong here — warning would be noise: %v", hits)
	}
}
