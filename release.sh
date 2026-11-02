#!/bin/sh
set -eu

# Usage: ./release.sh <patch|minor|major> [--push]
# Bumps version, runs checks, commits, and creates an annotated git tag.
# Default: no push, unless --push is specified.

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
cd "$SCRIPT_DIR"

PUSH=false
BUMP_TARGET=""

for arg in "$@"; do
  case "$arg" in
    --push)
      PUSH=true
      ;;
    -h|--help)
      echo "Usage: $0 <patch|minor|major> [--push]"
      echo ""
      echo "Arguments:"
      echo "  patch|minor|major  Bump semantic version component"
      echo ""
      echo "Options:"
      echo "  --push             Push bump commit and tag to git remote"
      exit 0
      ;;
    *)
      if [ -z "$BUMP_TARGET" ]; then
        BUMP_TARGET="$arg"
      else
        echo "Error: Unknown argument '$arg'" >&2
        echo "Usage: $0 <patch|minor|major> [--push]" >&2
        exit 1
      fi
      ;;
  esac
done

case "$BUMP_TARGET" in
  patch|minor|major)
    ;;
  "")
    echo "Error: Bump target is required. Expected: patch, minor, or major." >&2
    echo "Usage: $0 <patch|minor|major> [--push]" >&2
    exit 1
    ;;
  *)
    echo "Error: Invalid bump target '$BUMP_TARGET'. Expected: patch, minor, or major." >&2
    echo "Usage: $0 <patch|minor|major> [--push]" >&2
    exit 1
    ;;
esac

# Ensure git working tree is clean
if [ -n "$(git status --porcelain)" ]; then
  echo "Error: Git working tree is dirty. Please commit or stash changes before releasing." >&2
  exit 1
fi

# Determine current version from VERSION file
if [ -f "VERSION" ]; then
  CURRENT_VERSION=$(tr -d ' \n\r' < VERSION)
else
  CURRENT_VERSION="0.1.0"
fi
CURRENT_VERSION="${CURRENT_VERSION#v}"

MAJOR=$(echo "$CURRENT_VERSION" | cut -d. -f1)
MINOR=$(echo "$CURRENT_VERSION" | cut -d. -f2)
PATCH=$(echo "$CURRENT_VERSION" | cut -d. -f3)

case "$BUMP_TARGET" in
  patch)
    NEW_VERSION="${MAJOR}.${MINOR}.$((PATCH + 1))"
    ;;
  minor)
    NEW_VERSION="${MAJOR}.$((MINOR + 1)).0"
    ;;
  major)
    NEW_VERSION="$((MAJOR + 1)).0.0"
    ;;
  *)
    echo "Error: Invalid bump target '$BUMP_TARGET'. Expected: patch, minor, or major." >&2
    echo "Usage: $0 <patch|minor|major> [--push]" >&2
    exit 1
    ;;
esac

if [ "$NEW_VERSION" = "$CURRENT_VERSION" ]; then
  echo "Error: Target version ($NEW_VERSION) is identical to current version ($CURRENT_VERSION)." >&2
  exit 1
fi

TAG="v${NEW_VERSION}"

# Check if tag already exists
if git rev-parse "$TAG" >/dev/null 2>&1; then
  echo "Error: Git tag '$TAG' already exists." >&2
  exit 1
fi

echo "Preparing release $TAG (from v${CURRENT_VERSION})..."

# Run lint and test suites
echo "Running linter and tests..."
./lint.sh
./test.sh

# Update VERSION file
printf '%s\n' "$NEW_VERSION" > VERSION

# Update cmd/root.go if version constant exists
if [ -f "cmd/root.go" ]; then
  sed "s/var Version = \".*\"/var Version = \"$NEW_VERSION\"/" cmd/root.go > cmd/root.go.tmp && mv cmd/root.go.tmp cmd/root.go
fi

# Commit and tag
git add VERSION cmd/root.go
git commit -m "chore(release): bump version to ${TAG}"
git tag -a "$TAG" -m "Release ${TAG}"

echo "Successfully tagged ${TAG} at commit $(git rev-parse --short HEAD)."

# Handle remote push
if [ "$PUSH" = true ]; then
  REMOTE="origin"
  if ! git remote | grep -q "^origin$"; then
    REMOTE=$(git remote | head -n1)
  fi

  if [ -z "$REMOTE" ]; then
    echo "Error: No git remote configured to push to." >&2
    exit 1
  fi

  echo "Pushing commit and tag ${TAG} to remote '${REMOTE}'..."
  git push "$REMOTE" HEAD
  git push "$REMOTE" "$TAG"
  echo "Push complete."
else
  echo ""
  echo "Default: changes not pushed to remote."
  echo "To push manually, run:"
  echo "  git push origin HEAD && git push origin ${TAG}"
fi
