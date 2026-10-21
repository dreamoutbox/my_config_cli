#!/bin/sh
# Fake git used by unit tests to exercise auth-retry behaviour.
# Logs each invocation, then fails with a 403 unless the credential helper was
# reset and GITHUB_TOKEN is absent.
set -eu

log="${FAKE_GIT_LOG:?FAKE_GIT_LOG is required}"
{
	printf 'call\n'
	printf 'args=%s\n' "$*"
	printf 'github_token=%s\n' "${GITHUB_TOKEN-}"
	printf 'git_askpass=%s\n' "${GIT_ASKPASS-}"
	printf 'terminal_prompt=%s\n' "${GIT_TERMINAL_PROMPT-}"
} >>"$log"

case "${FAKE_GIT_MODE:-auth}" in
generic)
	echo "fatal: some unrelated failure" >&2
	exit 128
	;;
notfound)
	echo "fatal: unable to access 'https://github.com/org/repo.git/': The requested URL returned error: 404" >&2
	exit 128
	;;
esac

case " $* " in
*" -c credential.helper= "*)
	if [ -n "${GITHUB_TOKEN-}" ]; then
		echo "fatal: unable to access 'https://github.com/org/repo.git/': The requested URL returned error: 403" >&2
		exit 128
	fi
	printf 'refs/heads/main\tabc123\n'
	exit 0
	;;
*)
	echo "fatal: unable to access 'https://github.com/org/repo.git/': The requested URL returned error: 403" >&2
	exit 128
	;;
esac
