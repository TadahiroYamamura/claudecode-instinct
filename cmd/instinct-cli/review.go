package main

import (
	"context"
	"fmt"
	"io"
	"text/tabwriter"
)

func execReviewList(ctx context.Context, repo Repository, cfg *InstinctConfig, out io.Writer) error {
	teamBranch := cfg.Dolt.TeamBranch
	if teamBranch == "" {
		teamBranch = defaultTeamBranch
	}
	rows, err := repo.ListReviewQueue(ctx, teamBranch)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		fmt.Fprintf(out, "0 instinct(s) in review_queue on %s\n", teamBranch)
		return nil
	}
	return printReviewQueue(rows, out)
}

func printReviewQueue(rows []ReviewQueueRow, w io.Writer) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tCONTENT\tTRIGGER\tDOMAIN\tOBS\tSCOPE\tBY")
	for _, r := range rows {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%d\t%s\t%s\n",
			r.InstinctID[:shortIDLen],
			truncate(r.Content, contentMaxRunes),
			r.TriggerDesc,
			r.Domain,
			r.ObservationCount,
			r.Scope,
			r.SubmittedBy,
		)
	}
	return tw.Flush()
}

func execReviewApprove(ctx context.Context, repo Repository, cfg *InstinctConfig,
	personalBranch, approvedBy string, ids []string, out io.Writer) error {

	teamBranch := cfg.Dolt.TeamBranch
	if teamBranch == "" {
		teamBranch = defaultTeamBranch
	}

	rows, err := repo.ListReviewQueue(ctx, teamBranch)
	if err != nil {
		return err
	}

	byShortID := make(map[string]ReviewQueueRow, len(rows))
	for _, r := range rows {
		if len(r.InstinctID) >= shortIDLen {
			byShortID[r.InstinctID[:shortIDLen]] = r
		}
	}

	var selected []ReviewQueueRow
	for _, id := range ids {
		prefix := id
		if len(prefix) > shortIDLen {
			prefix = prefix[:shortIDLen]
		}
		if r, ok := byShortID[prefix]; ok {
			selected = append(selected, r)
		}
	}

	if len(selected) == 0 {
		return nil
	}

	if err := repo.PromoteFromReviewQueue(ctx, teamBranch, selected, personalBranch, approvedBy); err != nil {
		return err
	}
	fmt.Fprintf(out, "promoted %d instinct(s) to %s\n", len(selected), teamBranch)
	return nil
}
