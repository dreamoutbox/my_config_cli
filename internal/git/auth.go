package git

import (
	"fmt"
	"os"
	"strings"
)

// stdinIsTerminal reports whether an interactive credential prompt is possible.
// It is a variable so tests can simulate a terminal session.
var stdinIsTerminal = func() bool {
	info, err := os.Stdin.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

// GitRunInteractive executes git attached to the user's terminal so git can
// prompt for credentials. stdout is captured for callers that parse it, while
// stdin and stderr (where prompts are written) stay connected to the terminal.
//
// extraEnv, when non-nil, replaces the process environment; pass
// cleanCredentialEnv() to strip ambient tokens. The caller must also prepend
// "-c credential.helper=" to args to disable configured helpers for that run.
func GitRunInteractive(dir string, extraEnv []string, args ...string) (string, error) {
	return runGit(dir, extraEnv, true, args...)
}

// RunWithAuthRetry runs a network git command and, when the default credentials
// are rejected, retries once interactively with ambient tokens and credential
// helpers disabled. Retried credentials are supplied to git directly and are
// never persisted by myconfig.
//
// Use only for network commands: pull, fetch, push, ls-remote.
func RunWithAuthRetry(dir string, args ...string) (string, error) {
	out, err := GitRun(dir, args...)
	if err == nil {
		return out, nil
	}

	if IsNotFoundError(err) {
		return out, fmt.Errorf("repository does not exist (or URL is wrong, or credentials cannot see it): %w", err)
	}

	if !IsAuthError(err) {
		return out, err
	}

	if !stdinIsTerminal() {
		return out, fmt.Errorf("%w\nhint: no interactive terminal; use a personal access token in --repo (https://<user>:<token>@host/repo.git)", err)
	}

	fmt.Fprintf(os.Stderr, "Access denied with default credentials; enter credentials for %s\n", hostFromError(err))
	retryArgs := append([]string{"-c", "credential.helper="}, args...)
	return GitRunInteractive(dir, cleanCredentialEnv(), retryArgs...)
}

// credentialEnvVars are stripped on the interactive retry so no ambient token
// or askpass helper is used.
var credentialEnvVars = map[string]bool{
	"GH_TOKEN":            true,
	"GITHUB_TOKEN":        true,
	"GIT_ASKPASS":         true,
	"SSH_ASKPASS":         true,
	"GIT_TERMINAL_PROMPT": true,
}

// cleanCredentialEnv returns the process environment without token/askpass
// variables, with terminal prompting explicitly enabled.
func cleanCredentialEnv() []string {
	env := make([]string, 0, len(os.Environ())+3)
	for _, kv := range os.Environ() {
		name := kv
		if idx := strings.IndexByte(kv, '='); idx >= 0 {
			name = kv[:idx]
		}
		if credentialEnvVars[strings.ToUpper(name)] {
			continue
		}
		env = append(env, kv)
	}
	return append(env, "GIT_TERMINAL_PROMPT=1", "GIT_ASKPASS=", "SSH_ASKPASS=")
}
