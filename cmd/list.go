package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/k1LoW/gh-wait/internal/rule"
	"github.com/mergestat/timediff"
	"github.com/spf13/cobra"
)

var jsonOutput bool

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List watch rules",
	Long: `List all watch rules registered on the gh-wait server.

By default, output is a human-readable table with columns:
  ID, URL, CONDITIONS, UNTIL, COUNT, INTERVAL, ACTIONS, STATUS, LAST_TRIGGERED_AT

Use --json to output the rules as a JSON array for programmatic use.

Rules have one of the following statuses:
  watching   — Actively polling for conditions.
  triggered  — Condition was met and action was executed.
  stopped    — Rule was terminated by an --until condition or --count limit.

The server must be running for this command to work.`,
	Example: `  # List all rules in table format
  gh wait list

  # List all rules as JSON
  gh wait list --json

  # Pipe JSON output to jq
  gh wait list --json | jq '.[] | select(.status == "watching")'`,
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
		fmt.Fprintln(w, "ID\tURL\tCONDITIONS\tUNTIL\tCOUNT\tINTERVAL\tACTIONS\tSTATUS\tLAST_TRIGGERED_AT")
		for _, r := range rules {
			untilStr := "-"
			if len(r.Until) > 0 {
				untilStr = strings.Join(r.Until, ",")
			}
			countStr := "-"
			if r.MaxCount > 0 {
				countStr = fmt.Sprintf("%d/%d", r.TriggerCount, r.MaxCount)
			} else if len(r.Until) > 0 {
				countStr = fmt.Sprintf("%d", r.TriggerCount)
			}
			intervalStr := r.Interval
			if intervalStr == "" {
				intervalStr = rule.DefaultIntervalStr
			}
			lastTriggeredStr := "-"
			if !r.LastTriggeredAt.IsZero() {
				lastTriggeredStr = timediff.TimeDiff(r.LastTriggeredAt)
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				r.ID, r.URL,
				strings.Join(r.Conditions, ","), untilStr, countStr, intervalStr, strings.Join(r.Actions, ","), r.Status, lastTriggeredStr)
		}
		return w.Flush()
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
}
