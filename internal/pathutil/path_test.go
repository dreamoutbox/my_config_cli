package pathutil

import (
	"path/filepath"
	"testing"
)

func TestExpand(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "home only",
			input: "~",
			want:  tmp,
		},
		{
			name:  "tilde with subpath",
			input: "~/test/dir",
			want:  filepath.Join(tmp, "test", "dir"),
		},
		{
			name:  "absolute path",
			input: "/tmp/custom/path",
			want:  "/tmp/custom/path",
		},
		{
			name:  "relative path",
			input: "foo/bar",
			want:  "foo/bar",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Expand(tc.input)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Expand(%q) err = %v, wantErr = %v", tc.input, err, tc.wantErr)
			}
			if got != tc.want {
				t.Fatalf("Expand(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestCollapse(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "home directory itself",
			input: tmp,
			want:  "~",
		},
		{
			name:  "file inside home",
			input: filepath.Join(tmp, ".bashrc"),
			want:  "~/.bashrc",
		},
		{
			name:  "subfolder inside home",
			input: filepath.Join(tmp, ".config", "fish"),
			want:  "~/.config/fish",
		},
		{
			name:  "already tilde path",
			input: "~/.gitconfig",
			want:  "~/.gitconfig",
		},
		{
			name:  "path outside home",
			input: "/var/log/syslog",
			want:  "/var/log/syslog",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Collapse(tc.input)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Collapse(%q) err = %v, wantErr = %v", tc.input, err, tc.wantErr)
			}
			if got != tc.want {
				t.Fatalf("Collapse(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
