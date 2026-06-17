package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tamnd/x-cli/x"
)

// newAuthCmd builds `tori auth import|status|logout`, which manages the x-cli
// session tori shares with the rest of the toolchain (spec §12). tori holds no
// API key — the only secret here is the user's own browser session (Tier 2), the
// auth_token and ct0 cookies, stored by the x-cli session package.
func newAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage the X session (Tier 2) shared with x-cli",
	}
	cmd.AddCommand(authImportCmd(), authStatusCmd(), authLogoutCmd())
	return cmd
}

func authImportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Store your auth_token and ct0 cookies for the session tier",
		Long: "Import your own X session cookies so tori can read what the free\n" +
			"tiers cannot (your bookmarks, deeper history). Find auth_token and\n" +
			"ct0 in your browser's cookies for x.com. They are stored locally,\n" +
			"never transmitted anywhere but to X itself.\n\n" +
			"  tori auth import --auth-token <...> --ct0 <...>\n\n" +
			"The X_AUTH_TOKEN / X_CT0 environment variables are read when the\n" +
			"flags are omitted.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			f := cmd.Flags()
			authToken, _ := f.GetString("auth-token")
			ct0, _ := f.GetString("ct0")
			handle, _ := f.GetString("handle")
			if authToken == "" {
				authToken = os.Getenv("X_AUTH_TOKEN")
			}
			if ct0 == "" {
				ct0 = os.Getenv("X_CT0")
			}
			authToken = strings.TrimSpace(authToken)
			ct0 = strings.TrimSpace(ct0)
			if authToken == "" || ct0 == "" {
				return fmt.Errorf("both --auth-token and --ct0 are required (or set X_AUTH_TOKEN and X_CT0)")
			}
			if err := x.SaveSession(x.Creds{AuthToken: authToken, CT0: ct0, Handle: strings.TrimPrefix(handle, "@")}); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "session imported")
			return nil
		},
	}
	f := cmd.Flags()
	f.String("auth-token", "", "the auth_token cookie from x.com")
	f.String("ct0", "", "the ct0 cookie from x.com")
	f.String("handle", "", "your @handle (optional, for display)")
	return cmd
}

func authStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show whether a session is stored",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			creds, ok := x.LoadSession()
			if !ok {
				fmt.Fprintln(out, "no session stored (Tier 0 syndication and --guest still work)")
				return nil
			}
			if creds.Handle != "" {
				fmt.Fprintf(out, "session stored for @%s\n", creds.Handle)
			} else {
				fmt.Fprintln(out, "session stored")
			}
			return nil
		},
	}
}

func authLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove the stored session",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := x.ForgetSession(); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "session removed")
			return nil
		},
	}
}
