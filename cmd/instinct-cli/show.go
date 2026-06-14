package main

import (
	"context"
	"fmt"
	"io"
)

func execShow(ctx context.Context, repo Repository, shortID string, w io.Writer) error {
	r, err := repo.GetInstinct(ctx, shortID)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "%s\n\n", r.Content)
	fmt.Fprintf(w, "[trigger]\n%s\n\n", r.TriggerDesc)
	fmt.Fprintf(w, "[meta]\n")
	fmt.Fprintf(w, "id: %s\n", r.ID)
	fmt.Fprintf(w, "domain: %s\n", r.Domain)
	fmt.Fprintf(w, "obs: %d\n", r.ObservationCount)
	fmt.Fprintf(w, "scope: %s\n", r.Scope)
	return nil
}
