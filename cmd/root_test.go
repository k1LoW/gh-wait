package cmd

import (
	"testing"
)

func TestParseGitHubURL(t *testing.T) {
	tests := []struct {
		name          string
		raw           string
		wantSub       string
		wantRepo      string
		wantNumber    int
		wantNormURL   string
		wantOK        bool
	}{
		{"PR URL", "https://github.com/owner/repo/pull/42", "pr", "owner/repo", 42, "https://github.com/owner/repo/pull/42", true},
		{"Issue URL", "https://github.com/owner/repo/issues/5", "issue", "owner/repo", 5, "https://github.com/owner/repo/issues/5", true},
		{"PR URL with trailing slash", "https://github.com/owner/repo/pull/42/", "pr", "owner/repo", 42, "https://github.com/owner/repo/pull/42", true},
		{"PR URL with extra path", "https://github.com/owner/repo/pull/42/files", "pr", "owner/repo", 42, "https://github.com/owner/repo/pull/42", true},
		{"GHE URL", "https://ghe.example.com/org/project/pull/100", "pr", "org/project", 100, "https://ghe.example.com/org/project/pull/100", true},
		{"Not a URL", "42", "", "", 0, "", false},
		{"Not a GitHub URL", "https://example.com/foo/bar", "", "", 0, "", false},
		{"Bad number", "https://github.com/owner/repo/pull/abc", "", "", 0, "", false},
		{"Workflow run URL", "https://github.com/owner/repo/actions/runs/23424874935", "workflow", "owner/repo", 23424874935, "https://github.com/owner/repo/actions/runs/23424874935", true},
		{"Workflow run URL with trailing slash", "https://github.com/owner/repo/actions/runs/12345/", "workflow", "owner/repo", 12345, "https://github.com/owner/repo/actions/runs/12345", true},
		{"GHE workflow run URL", "https://ghe.example.com/org/project/actions/runs/99999", "workflow", "org/project", 99999, "https://ghe.example.com/org/project/actions/runs/99999", true},
		{"Unknown kind", "https://github.com/owner/repo/actions/123", "", "", 0, "", false},
		{"Zero number", "https://github.com/owner/repo/pull/0", "", "", 0, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub, repo, num, normURL, ok := parseGitHubURL(tt.raw)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if sub != tt.wantSub {
				t.Errorf("subcommand = %q, want %q", sub, tt.wantSub)
			}
			if repo != tt.wantRepo {
				t.Errorf("repo = %q, want %q", repo, tt.wantRepo)
			}
			if num != tt.wantNumber {
				t.Errorf("number = %d, want %d", num, tt.wantNumber)
			}
			if normURL != tt.wantNormURL {
				t.Errorf("normalizedURL = %q, want %q", normURL, tt.wantNormURL)
			}
		})
	}
}

func TestTransformURLArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantArgs []string
		wantOK   bool
	}{
		{
			"PR URL with flags",
			[]string{"https://github.com/owner/repo/pull/42", "--approved", "--open"},
			[]string{"pr", "42", "--repo", "owner/repo", "--url", "https://github.com/owner/repo/pull/42", "--approved", "--open"},
			true,
		},
		{
			"Issue URL with flags",
			[]string{"https://github.com/owner/repo/issues/5", "--commented"},
			[]string{"issue", "5", "--repo", "owner/repo", "--url", "https://github.com/owner/repo/issues/5", "--commented"},
			true,
		},
		{
			"URL with preceding global flags",
			[]string{"--port", "8080", "https://github.com/owner/repo/pull/10", "--merged"},
			[]string{"--port", "8080", "pr", "10", "--repo", "owner/repo", "--url", "https://github.com/owner/repo/pull/10", "--merged"},
			true,
		},
		{
			"GHE URL preserves host",
			[]string{"https://ghe.example.com/org/project/pull/7", "--approved"},
			[]string{"pr", "7", "--repo", "org/project", "--url", "https://ghe.example.com/org/project/pull/7", "--approved"},
			true,
		},
		{
			"Workflow run URL with flags",
			[]string{"https://github.com/owner/repo/actions/runs/23424874935", "--failed", "--open"},
			[]string{"workflow", "23424874935", "--repo", "owner/repo", "--url", "https://github.com/owner/repo/actions/runs/23424874935", "--failed", "--open"},
			true,
		},
		{
			"Subcommand (not a URL)",
			[]string{"pr", "42", "--approved"},
			nil,
			false,
		},
		{
			"No args",
			[]string{},
			nil,
			false,
		},
		{
			"Boolean flag before URL",
			[]string{"--foreground", "https://github.com/owner/repo/pull/7", "--approved"},
			[]string{"--foreground", "pr", "7", "--repo", "owner/repo", "--url", "https://github.com/owner/repo/pull/7", "--approved"},
			true,
		},
		{
			"Only flags",
			[]string{"--foreground"},
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := transformURLArgs(tt.args)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if len(got) != len(tt.wantArgs) {
				t.Fatalf("args = %v, want %v", got, tt.wantArgs)
			}
			for i := range got {
				if got[i] != tt.wantArgs[i] {
					t.Errorf("args[%d] = %q, want %q", i, got[i], tt.wantArgs[i])
				}
			}
		})
	}
}
