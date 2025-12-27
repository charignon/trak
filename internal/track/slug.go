// Package track provides slug generation and track status types for trak.
package track

import (
	"regexp"
	"strings"
	"unicode"
)

// nonAlphanumericRegex matches any character that is not alphanumeric or hyphen.
var nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9-]+`)

// multipleHyphensRegex matches multiple consecutive hyphens.
var multipleHyphensRegex = regexp.MustCompile(`-+`)

// Slugify converts a branch name to a URL-safe slug.
// Example: "feature/foo-bar" -> "feature-foo-bar"
// Example: "fix/auth bug" -> "fix-auth-bug"
// Example: "Feature/FOO" -> "feature-foo"
func Slugify(branch string) string {
	// Convert to lowercase
	slug := strings.ToLower(branch)

	// Replace common separators with hyphens
	slug = strings.ReplaceAll(slug, "/", "-")
	slug = strings.ReplaceAll(slug, "_", "-")
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, ".", "-")
	slug = strings.ReplaceAll(slug, "@", "-")
	slug = strings.ReplaceAll(slug, "#", "-")

	// Remove any remaining non-alphanumeric characters (except hyphens)
	slug = nonAlphanumericRegex.ReplaceAllString(slug, "")

	// Collapse multiple hyphens into single hyphen
	slug = multipleHyphensRegex.ReplaceAllString(slug, "-")

	// Trim leading and trailing hyphens
	slug = strings.Trim(slug, "-")

	return slug
}

// GenerateSlug creates a worktree directory name from branch and SHA.
// Example: "feature/foo", "a1b2c3d4e5f6" -> "feature-foo-a1b2c3d"
func GenerateSlug(branch, sha string) string {
	slug := Slugify(branch)

	// Take first 7 characters of SHA (standard short SHA)
	shortSHA := sha
	if len(sha) > 7 {
		shortSHA = sha[:7]
	}

	if shortSHA == "" {
		return slug
	}

	return slug + "-" + shortSHA
}

// SanitizeForTmux converts a branch name to a valid tmux window name.
// Tmux window names cannot contain: period (.), colon (:)
// Example: "feature/foo.bar" -> "feature/foo_bar"
// Example: "fix:auth" -> "fix_auth"
func SanitizeForTmux(branch string) string {
	// Replace problematic characters with underscores
	result := strings.Map(func(r rune) rune {
		switch r {
		case '.', ':':
			return '_'
		default:
			return r
		}
	}, branch)

	// Also handle any non-printable characters
	result = strings.Map(func(r rune) rune {
		if !unicode.IsPrint(r) {
			return '_'
		}
		return r
	}, result)

	// Ensure the name is not empty
	if result == "" {
		return "unnamed"
	}

	return result
}
