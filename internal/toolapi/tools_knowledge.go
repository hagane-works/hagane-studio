package toolapi

import (
	"context"
	"strings"

	"haganeworks/internal/knowledge"
)

type searchKnowledgeIn struct {
	Query  string `json:"query" jsonschema:"natural-language question, HOI4 script token, or authoring goal"`
	Locale string `json:"locale,omitempty" jsonschema:"auto, zh, en, or all; auto follows the query language"`
	Kind   string `json:"kind,omitempty" jsonschema:"optional article kind: tutorial, topic, guide, case, or reference"`
	Limit  int    `json:"limit,omitempty" jsonschema:"1 to 20; default 8"`
}

type getKnowledgeArticleIn struct {
	ID     string `json:"id" jsonschema:"exact id returned by search_knowledge; a slug or reference key is also accepted"`
	Locale string `json:"locale,omitempty" jsonschema:"zh or en; default zh with fallback to the available translation"`
}

func registerKnowledgeTools(server *Server) {
	AddTool(server, &Tool{
		Name:        "search_knowledge",
		Description: "Search the embedded offline HOI4 authoring knowledge base. Proactively use this before answering or implementing questions about HOI4 behavior, script placement, scopes, effects, triggers, modifiers, focuses, events, decisions, ideas, characters, technologies, localisation, or troubleshooting. Read the most relevant result with get_knowledge_article before writing. Results retain their verification level; source-checked guidance is not game-tested authority.",
	}, func(_ context.Context, _ *CallToolRequest, in searchKnowledgeIn) (*CallToolResult, map[string]any, error) {
		hits, total, err := knowledge.Search(knowledge.SearchOptions{Query: in.Query, Locale: in.Locale, Kind: in.Kind, Limit: in.Limit})
		if err != nil {
			return nil, nil, err
		}
		results := make([]map[string]any, 0, len(hits))
		for _, hit := range hits {
			article := hit.Article
			results = append(results, map[string]any{
				"id": article.ID, "kind": article.Kind, "slug": article.Slug, "locale": article.Locale,
				"title": article.Title, "summary": article.Summary, "topics": article.Topics,
				"scenarios": article.Scenarios, "verification": article.Verification,
				"game_version": article.GameVersion, "updated_at": article.UpdatedAt, "score": hit.Score,
			})
		}
		return nil, map[string]any{
			"query": strings.TrimSpace(in.Query), "matched": total, "returned": len(results), "results": results,
			"next": "Call get_knowledge_article with a relevant result id before relying on its guidance.",
		}, nil
	})

	AddTool(server, &Tool{
		Name:        "get_knowledge_article",
		Description: "Read one complete embedded HOI4 authoring article returned by search_knowledge. Use its verification and game_version when judging claims, reconcile examples with the current tool field reference, and do not present source-checked material as game-tested.",
	}, func(_ context.Context, _ *CallToolRequest, in getKnowledgeArticleIn) (*CallToolResult, map[string]any, error) {
		article, err := knowledge.Get(in.ID, in.Locale)
		if err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{
			"id": article.ID, "kind": article.Kind, "slug": article.Slug, "locale": article.Locale,
			"title": article.Title, "summary": article.Summary, "primary_topic": article.PrimaryTopic,
			"topics": article.Topics, "scenarios": article.Scenarios, "game_terms": article.GameTerms,
			"aliases": article.Aliases, "related_ids": article.RelatedIDs, "verification": article.Verification,
			"game_version": article.GameVersion, "updated_at": article.UpdatedAt,
			"reference_key": article.ReferenceKey, "sources": article.Sources, "body": article.Body,
		}, nil
	})
}
