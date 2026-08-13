package modelcatalog

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gaborltd/gac/internal/provider"
)

const DefaultURL = "https://raw.githubusercontent.com/gaborltd/gac/main/models.json"

//go:embed models.json
var embeddedFS embed.FS

type Catalog struct {
	SchemaVersion int                        `json:"schema_version"`
	UpdatedAt     string                     `json:"updated_at"`
	SourcePolicy  string                     `json:"source_policy"`
	Providers     map[string]ProviderCatalog `json:"providers"`
}

type ProviderCatalog struct {
	DocsURL   string         `json:"docs_url"`
	LoginHint string         `json:"login_hint"`
	Models    []CatalogModel `json:"models"`
}

type CatalogModel struct {
	ID              string `json:"id"`
	Value           string `json:"value"`
	DisplayName     string `json:"display_name"`
	CostTier        string `json:"cost_tier"`
	Recommended     bool   `json:"recommended"`
	ReasoningEffort string `json:"reasoning_effort"`
	Availability    string `json:"availability"`
	Documentation   string `json:"documentation"`
}

// Models 回傳推薦模型；chooser 使用 display name，最後再解析成 provider value。
func (c Catalog) Models(name string) []provider.Model {
	entry, ok := c.Providers[name]
	if !ok {
		return nil
	}
	models := make([]provider.Model, 0, len(entry.Models))
	for _, item := range entry.Models {
		if strings.TrimSpace(item.ID) == "" || !item.Recommended {
			continue
		}
		models = append(models, provider.Model{ID: item.ID, DisplayName: item.DisplayName, ProviderValue: item.Value, ReasoningEffort: item.ReasoningEffort})
	}
	return models
}

// ResolveValue 將清單顯示值轉成 provider CLI 真正接受的值。
func (c Catalog) ResolveValue(name, value string) string {
	for _, item := range c.Providers[name].Models {
		if value == item.ID || value == item.DisplayName || value == item.Value {
			return item.Value
		}
	}
	return value
}

func LoadEmbedded() (Catalog, error) {
	b, err := embeddedFS.ReadFile("models.json")
	if err != nil {
		return Catalog{}, fmt.Errorf("read embedded model catalog: %w", err)
	}
	return Parse(b)
}

// LoadRemote 優先讀取 GitHub raw 檔案，失敗時使用 binary 內建清單。
func LoadRemote(ctx context.Context, url string) (Catalog, error) {
	if strings.TrimSpace(url) == "" {
		url = DefaultURL
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err == nil {
		client := &http.Client{}
		response, requestErr := client.Do(request)
		if requestErr == nil {
			defer response.Body.Close()
			if response.StatusCode >= 200 && response.StatusCode < 300 {
				body, readErr := io.ReadAll(io.LimitReader(response.Body, 256*1024))
				if readErr == nil {
					catalog, parseErr := Parse(body)
					if parseErr == nil {
						return catalog, nil
					}
				}
			}
		}
	}
	return LoadEmbedded()
}

func Parse(data []byte) (Catalog, error) {
	var catalog Catalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return Catalog{}, fmt.Errorf("parse model catalog: %w", err)
	}
	if catalog.SchemaVersion != 1 {
		return Catalog{}, fmt.Errorf("unsupported model catalog schema: %d", catalog.SchemaVersion)
	}
	if len(catalog.Providers) == 0 {
		return Catalog{}, fmt.Errorf("model catalog has no providers")
	}
	return catalog, nil
}
