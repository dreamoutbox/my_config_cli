package git

import (
	"net/url"
	"regexp"
	"strings"
)

// errorPattern is a lowercase stderr substring used to classify git failures.
type errorPattern string

const (
	patternUnableToAccess     errorPattern = "unable to access"
	patternReturned403        errorPattern = "returned error: 403"
	patternAuthentication     errorPattern = "authentication failed"
	patternCouldNotReadUser   errorPattern = "could not read username"
	patternWriteAccessDenied  errorPattern = "write access to repository not granted"
	patternReturned404        errorPattern = "returned error: 404"
	patternRepositoryNotFound errorPattern = "repository not found"
)

var authErrorPatterns = []errorPattern{
	patternUnableToAccess,
	patternReturned403,
	patternAuthentication,
	patternCouldNotReadUser,
	patternWriteAccessDenied,
}

var notFoundErrorPatterns = []errorPattern{
	patternReturned404,
	patternRepositoryNotFound,
}

// IsAuthError reports whether err looks like a rejected-credential failure.
// A missing repository is never treated as an auth error, even though git
// often prefixes it with "unable to access".
func IsAuthError(err error) bool {
	if err == nil || IsNotFoundError(err) {
		return false
	}
	return matchesAny(err, authErrorPatterns)
}

// IsNotFoundError reports whether err indicates that the repository does not
// exist or is not visible to the current credentials (GitHub returns 404 for
// private repositories a token cannot access).
func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	return matchesAny(err, notFoundErrorPatterns)
}

func matchesAny(err error, patterns []errorPattern) bool {
	msg := strings.ToLower(err.Error())
	for _, pattern := range patterns {
		if strings.Contains(msg, string(pattern)) {
			return true
		}
	}
	return false
}

// urlUserinfoPattern matches userinfo in http(s) URLs, e.g. the "user:token@"
// part of https://user:token@github.com/org/repo.git.
var urlUserinfoPattern = regexp.MustCompile(`(?i)(https?://)[^/@\s]+@`)

// MaskURL removes embedded credentials from http(s) URLs so they are safe to
// print. SSH/scp-style URLs contain no secret and are returned unchanged.
//
// Example: MaskURL("https://u:tok@github.com/a/b.git") == "https://***@github.com/a/b.git"
func MaskURL(raw string) string {
	return urlUserinfoPattern.ReplaceAllString(raw, "${1}***@")
}

// urlPattern finds the first http(s) URL in a git error message.
var urlPattern = regexp.MustCompile(`(?i)https?://[^\s'"]+`)

// hostFromError extracts the remote host mentioned in a git error, falling
// back to a generic label when no URL is present.
func hostFromError(err error) string {
	match := urlPattern.FindString(err.Error())
	if match == "" {
		return "remote repository"
	}
	parsed, parseErr := url.Parse(strings.TrimRight(match, ".,;:)"))
	if parseErr != nil || parsed.Host == "" {
		return "remote repository"
	}
	return parsed.Host
}
