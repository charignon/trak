package track

// TrackType represents the type of track (worktree or devbox).
// Note: This duplicates db.TrackType for use in packages that don't need DB access.
type TrackType string

const (
	TrackTypeWorktree TrackType = "worktree"
	TrackTypeDevbox   TrackType = "devbox"
)

// GitStatus represents the git state of a track relative to its remote.
type GitStatus struct {
	Clean       bool // No uncommitted changes
	Ahead       bool // Local has commits not on remote
	Behind      bool // Remote has commits not in local
	AheadCount  int  // Number of commits ahead of remote
	BehindCount int  // Number of commits behind remote
}

// String returns a human-readable representation of the git status.
// Examples: "clean", "↑2", "↓3", "↑2↓3", "dirty"
func (s GitStatus) String() string {
	if !s.Clean {
		return "dirty"
	}

	if !s.Ahead && !s.Behind {
		return "clean"
	}

	result := ""
	if s.Ahead {
		result += "↑" + itoa(s.AheadCount)
	}
	if s.Behind {
		result += "↓" + itoa(s.BehindCount)
	}
	return result
}

// TrackStatus represents the full status of a track including git, PR, CI, and review states.
type TrackStatus struct {
	GitStatus   GitStatus
	PR          *PRStatus   // nil if no PR exists
	CI          *CIStatus   // nil if no CI configured or no PR
	Review      *ReviewStatus // nil if no PR
	IsStale     bool        // True if track hasn't been accessed recently
	SHAMismatch bool        // True if local SHA doesn't match expected (force-push detected)
}

// PRStatus represents the state of a pull request.
type PRStatus struct {
	Number int
	URL    string
	State  string // "open", "closed", "merged"
	Draft  bool
}

// CIStatus represents the CI/CD status of a pull request.
type CIStatus struct {
	Passing bool
	Pending bool
	Failing bool
	URL     string // Link to CI details
}

// CISymbol returns a symbol representing the CI status.
// Returns: "✓" for passing, "○" for pending, "✗" for failing, "—" for unknown
func (s *CIStatus) Symbol() string {
	if s == nil {
		return "—"
	}
	if s.Passing {
		return "✓"
	}
	if s.Pending {
		return "○"
	}
	if s.Failing {
		return "✗"
	}
	return "—"
}

// ReviewStatus represents the code review status of a pull request.
type ReviewStatus struct {
	Approved         bool
	ChangesRequested bool
	Pending          bool
	ReviewerCount    int
}

// ReviewSymbol returns a symbol representing the review status.
// Returns: "✓" for approved, "✗" for changes requested, "○" for pending, "—" for no reviews
func (s *ReviewStatus) Symbol() string {
	if s == nil {
		return "—"
	}
	if s.Approved {
		return "✓"
	}
	if s.ChangesRequested {
		return "✗"
	}
	if s.Pending {
		return "○"
	}
	return "—"
}

// itoa converts an integer to a string without importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}

	negative := n < 0
	if negative {
		n = -n
	}

	// Build digits in reverse
	digits := make([]byte, 0, 10)
	for n > 0 {
		digits = append(digits, byte('0'+n%10))
		n /= 10
	}

	// Reverse
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}

	if negative {
		return "-" + string(digits)
	}
	return string(digits)
}
