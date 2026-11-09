#!/usr/bin/env bash
# Create a new GOROCK + gorock-kit service from templates/skeleton.
#
# Usage: scaffold.sh <module-path> [target-dir]
#   scaffold.sh github.com/acme/orders ./orders
#
# Copies the skeleton, sets the module path, adds AGENTS.md, fetches the kit
# modules and builds. Refuses to write into a non-empty directory.
set -euo pipefail

if [ "$#" -lt 1 ]; then
  echo "usage: $0 <module-path> [target-dir]" >&2
  exit 2
fi

module="$1"
target="${2:-./$(basename "$module")}"
skill_dir="$(cd "$(dirname "$0")/.." && pwd)"
skeleton="$skill_dir/templates/skeleton"

case "$module" in
  *" "*|"" ) echo "invalid module path: '$module'" >&2; exit 2 ;;
esac

if [ -d "$target" ] && [ -n "$(ls -A "$target" 2>/dev/null)" ]; then
  echo "target directory '$target' is not empty; refusing to overwrite" >&2
  exit 1
fi

command -v go >/dev/null || { echo "go is not installed" >&2; exit 1; }

mkdir -p "$target"
cp -R "$skeleton/." "$target/"
cp "$skill_dir/templates/AGENTS.md" "$target/AGENTS.md"
mv "$target/gitignore" "$target/.gitignore"

# Replace the module placeholder in every text file.
find "$target" -type f \( -name '*.go' -o -name 'go.mod' \) -print0 |
  while IFS= read -r -d '' f; do
    sed -i.bak "s#__MODULE__#${module}#g" "$f" && rm -f "$f.bak"
  done

cd "$target"
export GOWORK=off
go get \
  github.com/arzab/gorock-kit/rockengine@latest \
  github.com/arzab/gorock-kit/rockconfig@latest \
  github.com/arzab/gorock-kit/rocklog@latest \
  github.com/arzab/gorock-kit/rockfiber@latest \
  github.com/gofiber/fiber/v2@latest
go mod tidy
gofmt -l . | (! grep .) || { echo "gofmt reported unformatted files" >&2; exit 1; }
go vet ./...
go build ./...

echo
echo "Created $module in $target"
echo "Run:  cd $target && go run ./cmd/main"
echo "Try:  curl 'localhost:8080/api/v1/hello?name=Ann'"
