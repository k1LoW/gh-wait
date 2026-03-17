package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var deleteAll bool

var deleteCmd = &cobra.Command{
	Use:   "delete [id...]",
	Short: "Delete watch rules",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !deleteAll && len(args) == 0 {
			return fmt.Errorf("requires at least 1 arg(s) or --all flag")
		}
		c := newClient()
		if deleteAll {
			rules, err := c.ListRules()
			if err != nil {
				return fmt.Errorf("failed to list rules: %w", err)
			}
			for _, r := range rules {
				if err := c.DeleteRule(r.ID); err != nil {
					return fmt.Errorf("failed to delete rule %s: %w", r.ID, err)
				}
				fmt.Printf("Deleted rule %s\n", r.ID)
			}
			return nil
		}
		for _, id := range args {
			if err := c.DeleteRule(id); err != nil {
				return fmt.Errorf("failed to delete rule %s: %w", id, err)
			}
			fmt.Printf("Deleted rule %s\n", id)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
	deleteCmd.Flags().BoolVar(&deleteAll, "all", false, "Delete all watch rules")
}
