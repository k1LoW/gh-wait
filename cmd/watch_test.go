package cmd

import (
	"slices"
	"testing"

	"github.com/spf13/cobra"
)

func TestParseWatchFlagsActions(t *testing.T) {
	tests := []struct {
		name    string
		open    bool
		notify  bool
		want    []string
	}{
		{"no flags", false, false, []string{"log"}},
		{"open only", true, false, []string{"open"}},
		{"notify only", false, true, []string{"notify"}},
		{"open and notify", true, true, []string{"open", "notify"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			registerWatchFlags(cmd)
			if tt.open {
				_ = cmd.Flags().Set("open", "true")
			}
			if tt.notify {
				_ = cmd.Flags().Set("notify", "true")
			}

			_, _, _, _, _, actions, err := parseWatchFlags(cmd, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !slices.Equal(actions, tt.want) {
				t.Errorf("actions = %v, want %v", actions, tt.want)
			}
		})
	}
}
