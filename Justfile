set dotenv-load

# List available commands.
default:
    @just --list

# Run the service locally using settings from .env.
dev:
    go run .

# Run all Go tests.
test:
    go test ./...

# Check Go code with vet.
lint:
    go vet ./...

# Compile all Go packages without leaving a binary in the checkout.
build:
    go build ./...

# Run the checks used before a release.
check: test lint build

# Start the Docker Compose service.
up:
    docker compose up -d

# Follow Docker Compose logs.
logs:
    docker compose logs -f

# Stop the Docker Compose service.
stop:
    docker compose stop

# Stop and remove the Docker Compose service.
down:
    docker compose down

# Bump the latest vMAJOR.MINOR.PATCH tag and push main and the tag together.
release level="patch":
    #!/usr/bin/env bash
    set -euo pipefail
    level={{quote(level)}}
    case "$level" in
      patch|minor|major) ;;
      *) printf 'Use: just release [patch|minor|major]\n' >&2; exit 1 ;;
    esac
    if [[ -n $(git status --porcelain) ]]; then
      printf 'Commit or discard all working-tree changes before releasing.\n' >&2
      exit 1
    fi
    if [[ $(git branch --show-current) != main ]]; then
      printf 'Release from the main branch.\n' >&2
      exit 1
    fi
    git fetch --tags origin main
    if ! git merge-base --is-ancestor origin/main HEAD; then
      printf 'Local main must contain the current origin/main. Pull or rebase first.\n' >&2
      exit 1
    fi
    latest=''
    while IFS= read -r candidate; do
      if [[ $candidate =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]]; then
        latest=$candidate
        break
      fi
    done < <(git tag --merged HEAD --list 'v*' --sort=-v:refname)
    if [[ -n $latest && $(git rev-list -n 1 "$latest") == $(git rev-parse HEAD) ]]; then
      printf 'HEAD is already tagged %s. Add a commit before releasing.\n' "$latest" >&2
      exit 1
    fi
    if [[ -z $latest ]]; then latest=v0.0.0; fi
    IFS=. read -r major minor patch <<< "${latest#v}"
    case "$level" in
      patch) patch=$((patch + 1)) ;;
      minor) minor=$((minor + 1)); patch=0 ;;
      major) major=$((major + 1)); minor=0; patch=0 ;;
    esac
    next="v${major}.${minor}.${patch}"
    if git show-ref --verify --quiet "refs/tags/$next"; then
      printf 'Tag %s already exists.\n' "$next" >&2
      exit 1
    fi
    just check
    if [[ -n $(git status --porcelain) ]]; then
      printf 'Checks changed the working tree; commit those changes first.\n' >&2
      exit 1
    fi
    git tag -a "$next" -m "Release $next"
    if ! git push --atomic origin HEAD:refs/heads/main "refs/tags/$next:refs/tags/$next"; then
      printf 'Push failed. Local tag %s remains; inspect the remote before retrying.\n' "$next" >&2
      exit 1
    fi
    printf 'Released %s\n' "$next"
