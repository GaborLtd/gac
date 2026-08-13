package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gaborltd/gac/internal/provider"
)

func TestChooseModelFromCatalogUsesProviderValue(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"schema_version":1,"providers":{"fake":{"models":[{"id":"cheap-id","value":"cheap-value","display_name":"Cheap","recommended":true,"reasoning_effort":"low"}]}}}`))
	}))
	defer server.Close()
	oldURL := os.Getenv("GAC_MODELS_URL")
	if err := os.Setenv("GAC_MODELS_URL", server.URL); err != nil {
		t.Fatal(err)
	}
	defer os.Setenv("GAC_MODELS_URL", oldURL)

	var out bytes.Buffer
	app := application{in: strings.NewReader("1\n"), out: &out, err: &bytes.Buffer{}}
	got, effort, err := app.chooseModelFromCatalog(context.Background(), fakeModelProvider{models: []provider.Model{{ID: "cheap-id", DisplayName: "Live Cheap", ProviderValue: "live-value"}}}, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "live-value" || effort != "low" {
		t.Fatalf("model = %q, want cheap-value", got)
	}
	if !strings.Contains(out.String(), "Recommended low-cost models") {
		t.Fatalf("catalog recommendation missing: %q", out.String())
	}
}

var _ provider.Provider = fakeModelProvider{}
