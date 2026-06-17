package cli

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tamnd/tori/repo"
)

// newInfoCmd builds `tori info <repo>`: a manifest summary — what the repo
// archives, how many records and media, the date range, the tiers used, the
// capture history, and the on-disk size.
func newInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info <repo>",
		Short: "Summarise a repository: counts, range, tiers, size",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := args[0]
			mf, ok, err := repo.LoadManifest(root)
			if err != nil {
				return err
			}
			if !ok || mf == nil {
				return fmt.Errorf("%s is not a tori repository (no manifest.json)", root)
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "repository: %s\n", root)
			fmt.Fprintf(out, "service:    %s\n", mf.Service)
			fmt.Fprintf(out, "target:     %s %s", mf.Target.Kind, mf.Target.Ref)
			if mf.Target.UserID != "" {
				fmt.Fprintf(out, " (id %s)", mf.Target.UserID)
			}
			fmt.Fprintln(out)
			fmt.Fprintf(out, "tweets:     %d\n", mf.Tweets)
			fmt.Fprintf(out, "threads:    %d\n", mf.Threads)
			fmt.Fprintf(out, "media:      %d local\n", mf.Media)
			if !mf.Range.Oldest.IsZero() {
				fmt.Fprintf(out, "range:      %s … %s\n",
					mf.Range.Oldest.UTC().Format("2006-01-02"),
					mf.Range.Newest.UTC().Format("2006-01-02"))
			}
			if len(mf.TiersUsed) > 0 {
				fmt.Fprintf(out, "tiers:      %s\n", strings.Join(mf.TiersUsed, ", "))
			}
			fmt.Fprintf(out, "captures:   %d\n", len(mf.Captures))
			for _, c := range mf.Captures {
				fmt.Fprintf(out, "  %s  +%d via %s\n", c.At, c.Added, c.Tier)
			}
			if size, err := dirSize(root); err == nil {
				fmt.Fprintf(out, "size:       %s\n", humanBytes(size))
			}
			return nil
		},
	}
}

// dirSize sums the size of every regular file under root.
func dirSize(root string) (int64, error) {
	var total int64
	err := filepath.WalkDir(root, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		total += info.Size()
		return nil
	})
	if os.IsNotExist(err) {
		return 0, nil
	}
	return total, err
}

func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for x := n / unit; x >= unit; x /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}
