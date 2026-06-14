package main

import (
	"context"
	"fmt"
	"io"
	"text/tabwriter"
)

const (
	contentMaxRunes = 40
	shortIDLen      = 8
)

func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "..."
}

func printInstincts(rows []InstinctRow, w io.Writer) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tCONTENT\tTRIGGER\tDOMAIN\tOBS\tSCOPE")
	for _, r := range rows {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%d\t%s\n",
			r.ID[:shortIDLen],
			truncate(r.Content, contentMaxRunes),
			r.TriggerDesc,
			r.Domain,
			r.ObservationCount,
			r.Scope,
		)
	}
	return tw.Flush()
}

func execListMerged(ctx context.Context, repo Repository, cfg *InstinctConfig, w io.Writer) error {
	teamBranch := cfg.Dolt.TeamBranch
	if teamBranch == "" {
		teamBranch = defaultTeamBranch
	}
	rows, err := repo.ListMergedInstincts(ctx, teamBranch)
	if err != nil {
		return err
	}
	return printInstincts(rows, w)
}

func execList(ctx context.Context, repo Repository, w io.Writer) error {
	rows, err := repo.ListInstincts(ctx)
	if err != nil {
		return err
	}
	return printInstincts(rows, w)
}

const defaultMediumThreshold = 6
