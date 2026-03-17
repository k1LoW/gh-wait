package cmd

import (
	"testing"
)

func TestParseGitHubURL(t *testing.T) {
	tests := []struct {
		name       string
		raw        string
		wantSub    string
		wantRepo   string
		wantNumber int
		wantOK     bool
	}{
		{"PR URL", "https://github.com/owner/repo/pull/42", "pr", "owner/repo", 42, true},
		{"Issue URL", "https://github.com/owner/repo/issues/5", "issue", "owner/repo", 5, true},
		{"PR URL with trailing slash", "https://github.com/owner/repo/pull/42/", "pr", "owner/repo", 42, true},
		{"PR URL with extra path", "https://github.com/owner/repo/pull/42/files", "pr", "owner/repo", 42, true},
		{"GHE URL", "https://ghe.example.com/org/project/pull/100", "pr", "org/project", 100, true},
		{"Not a URL", "42", "", "", 0, false},
		{"Not a GitHub URL", "https://example.com/foo/bar", "", "", 0, false},
		{"Bad number", "https://github.com/owner/repo/pull/abc", "", "", 0, false},
		{"Unknown kind", "https://github.com/owner/repo/actions/123", "", "", 0, false},
		{"Zero number", "https://github.com/owner/repo/pull/0", "", "", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub, repo, num, ok := parseGitHubURL(tt.raw)
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
			[]string{"pr", "42", "--repo", "owner/repo", "--approved", "--open"},
			true,
		},
		{
			"Issue URL with flags",
			[]string{"https://github.com/owner/repo/issues/5", "--commented"},
			[]string{"issue", "5", "--repo", "owner/repo", "--commented"},
			true,
		},
		{
			"URL with preceding global flags",
			[]string{"--port", "8080", "https://github.com/owner/repo/pull/10", "--merged"},
			[]string{"--port", "8080", "pr", "10", "--repo", "owner/repo", "--merged"},
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
