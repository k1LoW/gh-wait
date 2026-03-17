package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete [id...]",
	Short: "Delete watch rules",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c := newClient()
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
}
