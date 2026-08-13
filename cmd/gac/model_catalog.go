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

func (app *application) chooseModelFromCatalog(ctx context.Context, p provider.Provider, current, currentEffort string) (string, string, error) {
	catalog, err := modelcatalog.LoadRemote(ctx, os.Getenv("GAC_MODELS_URL"))
	if err != nil {
		fmt.Fprintf(app.err, "Warning: unable to load model catalog: %v\n", err)
		model, effort, err := app.chooseModelSelection(ctx, p, current, currentEffort)
		return model, effort, err
	}
	recommended := catalog.Models(p.Name())
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
