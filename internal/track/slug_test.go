package track

import "testing"

func TestSlugify(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		// Basic transformations
		{"simple branch", "main", "main"},
		{"feature branch", "feature/foo", "feature-foo"},
		{"nested feature", "feature/foo/bar", "feature-foo-bar"},
		{"with underscores", "fix_auth_bug", "fix-auth-bug"},
		{"mixed separators", "feature/foo_bar", "feature-foo-bar"},

		// Case handling
		{"uppercase", "FEATURE/FOO", "feature-foo"},
		{"mixed case", "Feature/Foo-Bar", "feature-foo-bar"},

		// Special characters
		{"with spaces", "fix auth bug", "fix-auth-bug"},
		{"with dots", "release.v1.0", "release-v1-0"},
		{"with at symbol", "user@feature", "user-feature"},
		{"with hash", "issue#123", "issue-123"},

		// Edge cases
		{"empty string", "", ""},
		{"only slashes", "///", ""},
		{"leading slash", "/feature", "feature"},
		{"trailing slash", "feature/", "feature"},
		{"multiple hyphens", "feature--foo---bar", "feature-foo-bar"},
		{"unicode", "feature/日本語", "feature"},

		// Real-world examples
		{"dependabot", "dependabot/npm_and_yarn/lodash-4.17.21", "dependabot-npm-and-yarn-lodash-4-17-21"},
		{"github issue ref", "fix/GH-123-auth-bug", "fix-gh-123-auth-bug"},
		{"jira ref", "PROJ-123/implement-feature", "proj-123-implement-feature"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Slugify(tt.input)
			if got != tt.expect {
				t.Errorf("Slugify(%q) = %q, want %q", tt.input, got, tt.expect)
			}
		})
	}
}

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		name   string
		branch string
		sha    string
		expect string
	}{
		{"basic", "feature/foo", "a1b2c3d4e5f6g7h8", "feature-foo-a1b2c3d"},
		{"short sha", "main", "abc", "main-abc"},
		{"exact 7 char sha", "main", "a1b2c3d", "main-a1b2c3d"},
		{"empty sha", "feature/bar", "", "feature-bar"},
		{"complex branch", "fix/auth-bug_v2", "deadbeef123", "fix-auth-bug-v2-deadbee"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateSlug(tt.branch, tt.sha)
			if got != tt.expect {
				t.Errorf("GenerateSlug(%q, %q) = %q, want %q", tt.branch, tt.sha, got, tt.expect)
			}
		})
	}
}

func TestSanitizeForTmux(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		// Basic - no changes needed
		{"simple", "main", "main"},
		{"with slash", "feature/foo", "feature/foo"},
		{"with hyphen", "fix-auth", "fix-auth"},

		// Characters that need replacing
		{"with period", "release.v1", "release_v1"},
		{"with colon", "fix:auth", "fix_auth"},
		{"multiple periods", "v1.2.3", "v1_2_3"},
		{"mixed bad chars", "foo.bar:baz", "foo_bar_baz"},

		// Edge cases
		{"empty string", "", "unnamed"},
		{"only periods", "...", "___"},
		{"real branch with version", "feature/v1.0.0-rc1", "feature/v1_0_0-rc1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeForTmux(tt.input)
			if got != tt.expect {
				t.Errorf("SanitizeForTmux(%q) = %q, want %q", tt.input, got, tt.expect)
			}
		})
	}
}

func TestGitStatusString(t *testing.T) {
	tests := []struct {
		name   string
		status GitStatus
		expect string
	}{
		{"clean", GitStatus{Clean: true}, "clean"},
		{"dirty", GitStatus{Clean: false}, "dirty"},
		{"ahead only", GitStatus{Clean: true, Ahead: true, AheadCount: 2}, "↑2"},
		{"behind only", GitStatus{Clean: true, Behind: true, BehindCount: 3}, "↓3"},
		{"ahead and behind", GitStatus{Clean: true, Ahead: true, Behind: true, AheadCount: 2, BehindCount: 3}, "↑2↓3"},
		{"zero counts", GitStatus{Clean: true, Ahead: false, Behind: false, AheadCount: 0, BehindCount: 0}, "clean"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.status.String()
			if got != tt.expect {
				t.Errorf("GitStatus.String() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestCIStatusSymbol(t *testing.T) {
	tests := []struct {
		name   string
		status *CIStatus
		expect string
	}{
		{"nil", nil, "—"},
		{"passing", &CIStatus{Passing: true}, "✓"},
		{"pending", &CIStatus{Pending: true}, "○"},
		{"failing", &CIStatus{Failing: true}, "✗"},
		{"unknown", &CIStatus{}, "—"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.status.Symbol()
			if got != tt.expect {
				t.Errorf("CIStatus.Symbol() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestReviewStatusSymbol(t *testing.T) {
	tests := []struct {
		name   string
		status *ReviewStatus
		expect string
	}{
		{"nil", nil, "—"},
		{"approved", &ReviewStatus{Approved: true}, "✓"},
		{"changes requested", &ReviewStatus{ChangesRequested: true}, "✗"},
		{"pending", &ReviewStatus{Pending: true}, "○"},
		{"no reviews", &ReviewStatus{}, "—"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.status.Symbol()
			if got != tt.expect {
				t.Errorf("ReviewStatus.Symbol() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestItoa(t *testing.T) {
	tests := []struct {
		input  int
		expect string
	}{
		{0, "0"},
		{1, "1"},
		{10, "10"},
		{123, "123"},
		{-1, "-1"},
		{-123, "-123"},
	}

	for _, tt := range tests {
		t.Run(tt.expect, func(t *testing.T) {
			got := itoa(tt.input)
			if got != tt.expect {
				t.Errorf("itoa(%d) = %q, want %q", tt.input, got, tt.expect)
			}
		})
	}
}
