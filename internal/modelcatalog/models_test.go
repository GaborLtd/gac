package modelcatalog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

const testCatalog = `{"schema_version":1,"updated_at":"2026-08-13","providers":{"fake":{"models":[{"id":"cheap-id","value":"cheap-value","display_name":"Cheap","cost_tier":"low","reasoning_effort":"low","recommended":true},{"id":"expensive-id","value":"expensive-value","display_name":"Expensive","cost_tier":"high","recommended":false}]}}}`

func TestParseReturnsOnlyRecommendedModelsAndProviderValue(t *testing.T) {
	catalog, err := Parse([]byte(testCatalog))
	if err != nil {
		t.Fatal(err)
	}
	models := catalog.Models("fake")
	if len(models) != 1 || models[0].ID != "cheap-id" || models[0].DisplayName != "Cheap" || catalog.ResolveValue("fake", models[0].DisplayName) != "cheap-value" || models[0].ReasoningEffort != "low" {
		t.Fatalf("models = %#v", models)
	}
}

func TestLoadRemoteUsesServerCatalog(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(testCatalog))
	}))
	defer server.Close()
	catalog, err := LoadRemote(context.Background(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Models("fake")) != 1 {
		t.Fatalf("catalog = %#v", catalog)
	}
}

func TestLoadRemoteFallsBackToEmbeddedCatalog(t *testing.T) {
	catalog, err := LoadRemote(context.Background(), "http://127.0.0.1:1/unavailable")
	if err != nil {
		t.Fatal(err)
	}
	models := catalog.Models("codex")
	if len(models) < 3 || models[0].ID != "gpt-5.4-mini" || models[0].ReasoningEffort != "low" {
		t.Fatalf("fallback catalog = %#v", models)
	}
}
