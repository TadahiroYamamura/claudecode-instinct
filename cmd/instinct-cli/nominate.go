package main

import (
	"context"
	"fmt"
	"io"
)

func execNominateList(ctx context.Context, repo Repository, cfg *InstinctConfig, out io.Writer) error {
	teamBranch := cfg.Dolt.TeamBranch
	if teamBranch == "" {
		teamBranch = defaultTeamBranch
	}
	minObs := cfg.Confidence.ReviewMin
	if minObs == 0 {
		minObs = defaultMediumThreshold
	}
	rows, err := repo.ListReviewInstincts(ctx, teamBranch, minObs)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		fmt.Fprintf(out, "0 instinct(s) eligible for nomination (all already on %s)\n", teamBranch)
		return nil
	}
	return printInstincts(rows, out)
}

func execNominate(ctx context.Context, repo Repository, cfg *InstinctConfig,
	personalBranch, submittedBy string, ids []string, out io.Writer) error {

	teamBranch := cfg.Dolt.TeamBranch
	if teamBranch == "" {
		teamBranch = defaultTeamBranch
	}
	minObs := cfg.Confidence.ReviewMin
	if minObs == 0 {
		minObs = defaultMediumThreshold
	}

	rows, err := repo.ListReviewInstincts(ctx, teamBranch, minObs)
	if err != nil {
		return err
	}

	byShortID := make(map[string]InstinctRow, len(rows))
	for _, r := range rows {
		if len(r.ID) >= shortIDLen {
			byShortID[r.ID[:shortIDLen]] = r
		}
	}

	var selected []InstinctRow
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

	if err := repo.SubmitToReviewQueue(ctx, teamBranch, selected, personalBranch, submittedBy); err != nil {
		return err
	}
	fmt.Fprintf(out, "nominated %d instinct(s) to review_queue on %s\n", len(selected), teamBranch)
	return nil
}
