package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tamnd/tori/archive"
)

// newRenderCmd builds `tori render <repo>`: re-render the HTML and Markdown views
// from the stored JSON with no network (spec §5, TP3). This is how a renderer
// improvement is replayed over an old archive, and how a Markdown view is added
// to an HTML-only repo.
func newRenderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "render <repo>",
		Short: "Re-render views from stored JSON, no network",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f := cmd.Flags()
			views, err := parseViews(mustString(f, "view"))
			if err != nil {
				return err
			}
			date, err := parseDate(mustString(f, "date"))
			if err != nil {
				return fmt.Errorf("parse --date: %w", err)
			}
			res, err := archive.Render(args[0], archive.RenderOptions{
				Views:   views,
				Date:    date,
				Version: Version,
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "rendered %d tweets, %d threads in %s\n", res.Total, res.Threads, args[0])
			return nil
		},
	}
	f := cmd.Flags()
	f.String("view", "html,md", "views to render: html|md|html,md")
	f.String("date", "", "fix the footer stamp (RFC3339) for reproducible output")
	return cmd
}
