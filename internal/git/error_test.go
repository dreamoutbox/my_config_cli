package git

import (
	"errors"
	"testing"
)

func TestMaskURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "PAT URL",
			in:   "https://user:ghp_secret@github.com/org/repo.git",
			want: "https://***@github.com/org/repo.git",
		},
		{
			name: "token only URL",
			in:   "https://ghp_secret@github.com/org/repo.git",
			want: "https://***@github.com/org/repo.git",
		},
		{
			name: "plain URL",
			in:   "https://github.com/org/repo.git",
			want: "https://github.com/org/repo.git",
		},
		{
			name: "scp-style ssh URL",
			in:   "git@github.com:org/repo.git",
			want: "git@github.com:org/repo.git",
		},
		{
			name: "ssh scheme URL",
			in:   "ssh://git@github.com/org/repo.git",
			want: "ssh://git@github.com/org/repo.git",
		},
		{
			name: "URL inside git error",
			in:   "fatal: unable to access 'https://u:tok@github.com/x/': 403",
			want: "fatal: unable to access 'https://***@github.com/x/': 403",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MaskURL(tt.in); got != tt.want {
				t.Fatalf("MaskURL(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestIsAuthError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "403", err: errors.New("fatal: unable to access 'https://github.com/x': The requested URL returned error: 403"), want: true},
		{name: "authentication failed", err: errors.New("fatal: Authentication failed for 'https://github.com/x'"), want: true},
		{name: "could not read username", err: errors.New("fatal: could not read Username for 'https://github.com': terminal prompts disabled"), want: true},
		{name: "write access denied", err: errors.New("remote: Write access to repository not granted."), want: true},
		{name: "404 is not auth", err: errors.New("fatal: unable to access 'https://github.com/x': The requested URL returned error: 404"), want: false},
		{name: "repository not found is not auth", err: errors.New("remote: Repository not found."), want: false},
		{name: "unrelated", err: errors.New("fatal: not a git repository"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsAuthError(tt.err); got != tt.want {
				t.Fatalf("IsAuthError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "404", err: errors.New("fatal: unable to access 'https://github.com/x': The requested URL returned error: 404"), want: true},
		{name: "repository not found", err: errors.New("remote: Repository not found."), want: true},
		{name: "403", err: errors.New("fatal: unable to access 'https://github.com/x': The requested URL returned error: 403"), want: false},
		{name: "unrelated", err: errors.New("fatal: not a git repository"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNotFoundError(tt.err); got != tt.want {
				t.Fatalf("IsNotFoundError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
