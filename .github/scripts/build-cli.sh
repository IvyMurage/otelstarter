#!/usr/bin/env bash
# Builds the otelstarter CLI for every supported platform and packs each one
# for a GitHub release:
#
#   .github/scripts/build-cli.sh v0.1.0 dist
#
# gives dist/otelstarter_linux_amd64.tar.gz, …_windows_amd64.zip and
# checksums.txt. Archive names carry no version, so
# https://github.com/IvyMurage/otelstarter/releases/latest/download/otelstarter_linux_amd64.tar.gz
# always points at the newest release.
set -euo pipefail

version="${1:?usage: build-cli.sh <version, e.g. v0.1.0> <output dir>}"
out="${2:?usage: build-cli.sh <version> <output dir>}"
root="$(cd "$(dirname "$0")/../.." && pwd)"

platforms=(
  linux/amd64 linux/arm64
  darwin/amd64 darwin/arm64
  windows/amd64 windows/arm64
)

rm -rf "$out"
mkdir -p "$out"
out="$(cd "$out" && pwd)"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

for platform in "${platforms[@]}"; do
  os="${platform%/*}"
  arch="${platform#*/}"
  name="otelstarter_${os}_${arch}"
  binary="otelstarter"
  [ "$os" = windows ] && binary="otelstarter.exe"

  mkdir -p "$work/$name"
  # CGO off: a static binary that runs without any system libraries.
  # -trimpath leaves the build machine's paths out; -s -w drops debug data.
  (cd "$root" && CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" go build -trimpath \
    -ldflags "-s -w -X github.com/IvyMurage/otelstarter/internal/cli.Version=$version" \
    -o "$work/$name/$binary" ./cmd/otelstarter)
  cp "$root/LICENSE" "$root/README.md" "$work/$name/"

  if [ "$os" = windows ]; then
    (cd "$work/$name" && zip -q "$out/$name.zip" otelstarter.exe LICENSE README.md)
  else
    tar -czf "$out/$name.tar.gz" -C "$work/$name" otelstarter LICENSE README.md
  fi
  echo "built $name"
done

(cd "$out" && sha256sum ./*.tar.gz ./*.zip | sed 's| \./| |' > checksums.txt)
echo "checksums.txt:"
cat "$out/checksums.txt"
