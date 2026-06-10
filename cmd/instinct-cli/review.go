package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

type reviewSelector func(rows []InstinctRow, out io.Writer) ([]string, error)

func execReview(ctx context.Context, conn *sql.Conn, cfg *InstinctConfig,
	personalBranch, submittedBy string, selector reviewSelector, out io.Writer) error {

	teamBranch := cfg.Dolt.TeamBranch
	if teamBranch == "" {
		teamBranch = defaultTeamBranch
	}
	minObs := cfg.Confidence.ReviewMin
	if minObs == 0 {
		minObs = defaultMediumThreshold
	}

	rows, err := listReviewInstincts(ctx, conn, teamBranch, minObs)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		fmt.Fprintf(out, "0 instinct(s) pending review (not yet on %s)\n", teamBranch)
		return nil
	}

	selectedIDs, err := selector(rows, out)
	if err != nil {
		return err
	}
	if len(selectedIDs) == 0 {
		return nil
	}

	byID := make(map[string]InstinctRow, len(rows))
	for _, r := range rows {
		byID[r.ID] = r
	}
	var selected []InstinctRow
	for _, id := range selectedIDs {
		if r, ok := byID[id]; ok {
			selected = append(selected, r)
		}
	}

	if err := submitToReviewQueue(ctx, conn, teamBranch, selected, personalBranch, submittedBy); err != nil {
		return err
	}
	fmt.Fprintf(out, "submitted %d instinct(s) to review_queue on %s\n", len(selected), teamBranch)
	return nil
}

func submitToReviewQueue(ctx context.Context, conn *sql.Conn, teamBranch string,
	rows []InstinctRow, personalBranch, submittedBy string) error {

	if _, err := conn.ExecContext(ctx, "CALL dolt_checkout(?)", teamBranch); err != nil {
		return fmt.Errorf("checkout %s: %w", teamBranch, err)
	}
	defer conn.ExecContext(ctx, "CALL dolt_checkout(?)", personalBranch) //nolint:errcheck

	for _, r := range rows {
		_, err := conn.ExecContext(ctx, `
			INSERT INTO review_queue
			  (instinct_id, content, trigger_desc, domain, observation_count, scope, submitted_by)
			VALUES (?, ?, ?, ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE
			  submitted_by = VALUES(submitted_by),
			  submitted_at = CURRENT_TIMESTAMP`,
			r.ID, r.Content, r.TriggerDesc, r.Domain, r.ObservationCount, r.Scope, submittedBy)
		if err != nil {
			return fmt.Errorf("insert review_queue %s: %w", r.ID[:shortIDLen], err)
		}
	}

	msg := fmt.Sprintf("review: submit %d instinct(s) by %s", len(rows), submittedBy)
	if _, err := conn.ExecContext(ctx, "CALL dolt_commit('-Am', ?)", msg); err != nil {
		// 全行が既存で変更なしの場合は正常終了
		if strings.Contains(err.Error(), "nothing to commit") {
			return nil
		}
		return fmt.Errorf("commit review_queue: %w", err)
	}
	return nil
}

func ttyReviewSelector(rows []InstinctRow, out io.Writer) ([]string, error) {
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
					ids = append(ids, r.ID)
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

func printReviewTUI(w io.Writer, rows []InstinctRow, cursor int, selected []bool, redraw bool) {
	const headerLines = 1
	if redraw {
		fmt.Fprintf(w, "\033[%dA\033[J", headerLines+len(rows))
	}
	fmt.Fprintf(w, "Review candidates (↑↓: navigate  space: toggle  enter: submit  q: quit)\r\n")
	for i, r := range rows {
		check := "[ ]"
		if selected[i] {
			check = "[x]"
		}
		arrow := "  "
		if i == cursor {
			arrow = "> "
		}
		fmt.Fprintf(w, "%s%s %s  %-38s %-12s %3d  %s\r\n",
			arrow, check, r.ID[:shortIDLen],
			truncate(r.Content, 38), r.Domain, r.ObservationCount, r.Scope,
		)
	}
}
