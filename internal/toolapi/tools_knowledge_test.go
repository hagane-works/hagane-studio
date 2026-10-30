package toolapi

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestKnowledgeToolsSearchAndRead(t *testing.T) {
	server := NewServer()
	registerKnowledgeTools(server)

	search, err := server.CallTool(context.Background(), "search_knowledge", json.RawMessage(`{"query":"国策奖励政治点","limit":3}`))
	if err != nil || search.IsError {
		t.Fatalf("search failed: %v %s", err, contentText(search))
	}
	out := search.StructuredContent.(map[string]any)
	results := out["results"].([]map[string]any)
	if len(results) == 0 || out["matched"].(int) == 0 {
		t.Fatalf("search result=%#v", out)
	}

	read, err := server.CallTool(context.Background(), "get_knowledge_article", json.RawMessage(`{"id":"wiki:effect:add_political_power","locale":"zh"}`))
	if err != nil || read.IsError {
		t.Fatalf("read failed: %v %s", err, contentText(read))
	}
	article := read.StructuredContent.(map[string]any)
	if article["verification"] == "" || !strings.Contains(article["body"].(string), "add_political_power") {
		t.Fatalf("article=%#v", article)
	}
}

func TestKnowledgeToolDescriptionsRequireProactiveLookup(t *testing.T) {
	server := NewServer()
	registerKnowledgeTools(server)
	tools, err := server.ListTools(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(tools) != 2 || tools[1].Name != "search_knowledge" || !strings.Contains(tools[1].Description, "Proactively") {
		t.Fatalf("tools=%+v", tools)
	}
}
