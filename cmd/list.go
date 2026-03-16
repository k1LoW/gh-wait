package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var jsonOutput bool

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List watch rules",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := newClient()
		rules, err := c.ListRules()
		if err != nil {
			return fmt.Errorf("failed to list rules (is the server running?): %w", err)
		}

		if jsonOutput {
			return json.NewEncoder(os.Stdout).Encode(rules)
		}

		if len(rules) == 0 {
			fmt.Println("No watch rules")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tTYPE\tREPO\tNUMBER\tCONDITIONS\tACTION\tSTATUS")
		for _, r := range rules {
			fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%s\t%s\n",
				r.ID, r.Type, r.Repo, r.Number,
				strings.Join(r.Conditions, ","), r.Action, r.Status)
		}
		return w.Flush()
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
}
