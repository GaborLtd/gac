package main

import (
	"context"
	"fmt"
	"os"

	"github.com/gaborltd/gac/internal/modelcatalog"
	"github.com/gaborltd/gac/internal/provider"
)

type recommendedModelProvider struct {
	provider.Provider
	models []provider.Model
}

func (p recommendedModelProvider) ListModels(context.Context) ([]provider.Model, error) {
	return p.models, nil
}

func reconcileRecommendedModels(ctx context.Context, p provider.Provider, recommended []provider.Model) []provider.Model {
	if len(recommended) == 0 {
		return recommended
	}
	live, err := p.ListModels(ctx)
	if err != nil || len(live) == 0 {
		return recommended
	}
	byID := make(map[string]provider.Model, len(live))
	for _, model := range live {
		byID[model.ID] = model
	}
	filtered := make([]provider.Model, 0, len(recommended))
	for _, model := range recommended {
		actual, ok := byID[model.ID]
		if !ok {
			continue
		}
		model.ProviderValue = actual.Value()
		if actual.DisplayName != "" {
			model.DisplayName = actual.DisplayName
		}
		filtered = append(filtered, model)
	}
	if len(filtered) > 0 {
		return filtered
	}
	return recommended
}

func (app *application) chooseModelFromCatalog(ctx context.Context, p provider.Provider, current, currentEffort string) (string, string, error) {
	catalog, err := modelcatalog.LoadRemote(ctx, os.Getenv("GAC_MODELS_URL"))
	if err != nil {
		fmt.Fprintf(app.err, "Warning: unable to load model catalog: %v\n", err)
		model, effort, err := app.chooseModelSelection(ctx, p, current, currentEffort)
		return model, effort, err
	}
	recommended := reconcileRecommendedModels(ctx, p, catalog.Models(p.Name()))
	if len(recommended) == 0 {
		model, effort, err := app.chooseModelSelection(ctx, p, current, currentEffort)
		return model, effort, err
	}
	fmt.Fprintln(app.out, "Recommended low-cost models loaded from the gac catalog.")
	fmt.Fprintln(app.out, "Availability and pricing may vary by provider account and CLI version.")
	selected, effort, err := app.chooseModelSelection(ctx, recommendedModelProvider{Provider: p, models: recommended}, current, currentEffort)
	if err != nil {
		return "", "", err
	}
	return catalog.ResolveValue(p.Name(), selected), effort, nil
}
