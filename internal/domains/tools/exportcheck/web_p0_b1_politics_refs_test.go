package exportcheck

import (
	"haganeworks/internal/hoi4/politicsrefs"
	"testing"
)

func TestParseIdeologyDefinitionsPreservesRootAndSubtypeIdentity(t *testing.T) {
	definitions, err := parseIdeologyDefinitions(`ideologies = {
		civic_coalition = { types = { civic_reformism = { can_be_randomly_selected = no } } color = { 1 2 3 } }
	}`)
	if err != nil {
		t.Fatal(err)
	}
	if len(definitions) != 2 || definitions[0].Kind != politicsrefs.KindRoot ||
		definitions[0].ScriptID != "civic_coalition" || definitions[1].Kind != politicsrefs.KindSubtype ||
		definitions[1].ScriptID != "civic_reformism" || definitions[1].RootID != "civic_coalition" || !definitions[1].RandomDisabled {
		t.Fatalf("definitions=%#v", definitions)
	}
}

func TestUnparseableScriptReferenceScanFailsClosed(t *testing.T) {
	known := map[string]politicsrefs.Ideology{
		"civic_coalition": {Kind: politicsrefs.KindRoot, ScriptID: "civic_coalition", Source: politicsrefs.SourceProjectManaged},
	}
	tokens := scriptIdeologyTokens(`available = { has_government = civic_coalition`, known)
	if len(tokens) != 1 || tokens[0].ID != "civic_coalition" || tokens[0].Context != "conservative token match in unparsed script" {
		t.Fatalf("tokens=%#v", tokens)
	}
}
