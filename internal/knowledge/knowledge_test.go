package knowledge

import (
	"strings"
	"testing"
)

func TestEmbeddedPublishedCorpus(t *testing.T) {
	articles, err := All()
	if err != nil {
		t.Fatal(err)
	}
	if len(articles) != 96 {
		t.Fatalf("articles=%d want=96", len(articles))
	}
	for _, article := range articles {
		if article.ID == "publication-draft-canary" || article.ID == "publication-archived-canary" {
			t.Fatalf("unpublished canary embedded: %s", article.ID)
		}
		if article.Body == "" || article.Title == "" || article.Verification == "" || len(article.Sources) == 0 {
			t.Fatalf("incomplete article: %+v", article)
		}
	}
}

func TestSearchKnowledgeByGoalAndToken(t *testing.T) {
	zh, total, err := Search(SearchOptions{Query: "完成国策获得150政治点", Limit: 5})
	if err != nil || total == 0 || len(zh) == 0 {
		t.Fatalf("zh search: total=%d hits=%v err=%v", total, zh, err)
	}
	if zh[0].Article.Locale != "zh" {
		t.Fatalf("auto locale=%q want zh", zh[0].Article.Locale)
	}
	found := false
	for _, hit := range zh {
		if hit.Article.ID == "focus-political-power-reward" || hit.Article.ID == "wiki:effect:add_political_power" {
			found = true
		}
	}
	if !found {
		t.Fatalf("political-power guidance missing: %+v", zh)
	}

	en, total, err := Search(SearchOptions{Query: "advisor unlocked after focus", Locale: "en", Limit: 5})
	if err != nil || total == 0 || len(en) == 0 {
		t.Fatalf("en search: total=%d hits=%v err=%v", total, en, err)
	}
	if en[0].Article.ID != "advisor-unlocked-by-focus" {
		t.Fatalf("top result=%q", en[0].Article.ID)
	}
}

func TestGetKnowledgeArticle(t *testing.T) {
	article, err := Get("effect/add_political_power", "zh")
	if err != nil {
		t.Fatal(err)
	}
	if article.ID != "wiki:effect:add_political_power" || !strings.Contains(article.Body, "add_political_power = 150") {
		t.Fatalf("article mismatch: %+v", article)
	}
	if _, err := Get("missing", "zh"); err == nil {
		t.Fatal("missing article did not fail")
	}
}

func TestSearchKnowledgeValidation(t *testing.T) {
	for _, options := range []SearchOptions{
		{},
		{Query: "focus", Locale: "de"},
		{Query: "focus", Limit: 21},
	} {
		if _, _, err := Search(options); err == nil {
			t.Fatalf("invalid options accepted: %+v", options)
		}
	}
}
