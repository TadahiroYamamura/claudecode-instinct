package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

type reviewSelector func(rows []ReviewQueueRow, out io.Writer) ([]string, error)

func execReview(ctx context.Context, repo Repository, cfg *InstinctConfig,
	personalBranch, approvedBy string, selector reviewSelector, out io.Writer) error {

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

	selectedIDs, err := selector(rows, out)
	if err != nil {
		return err
	}
	if len(selectedIDs) == 0 {
		return nil
	}

	byID := make(map[string]ReviewQueueRow, len(rows))
	for _, r := range rows {
		byID[r.InstinctID] = r
	}
	var selected []ReviewQueueRow
	for _, id := range selectedIDs {
		if r, ok := byID[id]; ok {
			selected = append(selected, r)
		}
	}

	if err := repo.PromoteFromReviewQueue(ctx, teamBranch, selected, personalBranch, approvedBy); err != nil {
		return err
	}
	fmt.Fprintf(out, "promoted %d instinct(s) to %s\n", len(selected), teamBranch)
	return nil
}

func ttyReviewSelector(rows []ReviewQueueRow, out io.Writer) ([]string, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return nil, nil
	}

	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return nil, fmt.Errorf("raw terminal: %w", err)
	}
	defer term.Restore(fd, oldState) //nolint:errcheck

	cursor := 0
	selected := make([]bool, len(rows))

	printReviewTUI(out, rows, cursor, selected, false)

	buf := make([]byte, 3)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			return nil, err
		}

		switch {
		case n == 1 && (buf[0] == 'q' || buf[0] == 3): // q or Ctrl-C
			fmt.Fprint(out, "\r\n")
			return nil, nil
		case n == 1 && buf[0] == '\r': // enter
			fmt.Fprint(out, "\r\n")
			var ids []string
			for i, r := range rows {
				if selected[i] {
					ids = append(ids, r.InstinctID)
				}
			}
			return ids, nil
		case n == 1 && buf[0] == ' ': // space: toggle
			selected[cursor] = !selected[cursor]
		case n == 3 && buf[0] == '\x1b' && buf[1] == '[' && buf[2] == 'A': // up
			if cursor > 0 {
				cursor--
			}
		case n == 3 && buf[0] == '\x1b' && buf[1] == '[' && buf[2] == 'B': // down
			if cursor < len(rows)-1 {
				cursor++
			}
		}

		printReviewTUI(out, rows, cursor, selected, true)
	}
}

func printReviewTUI(w io.Writer, rows []ReviewQueueRow, cursor int, selected []bool, redraw bool) {
	const headerLines = 1
	if redraw {
		fmt.Fprintf(w, "\033[%dA\033[J", headerLines+len(rows))
	}
	fmt.Fprintf(w, "Review queue (↑↓: navigate  space: toggle  enter: promote  q: quit)\r\n")
	for i, r := range rows {
		check := "[ ]"
		if selected[i] {
			check = "[x]"
		}
		arrow := "  "
		if i == cursor {
			arrow = "> "
		}
		fmt.Fprintf(w, "%s%s %s  %-38s %-20s %-12s %3d  %s  %s\r\n",
			arrow, check, r.InstinctID[:shortIDLen],
			truncate(r.Content, 38), truncate(r.TriggerDesc, 20), r.Domain, r.ObservationCount, r.Scope, r.SubmittedBy,
		)
	}
}
