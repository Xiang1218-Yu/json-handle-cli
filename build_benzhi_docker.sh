#!/usr/bin/env bash
set -euo pipefail

for platform in linux/amd64 linux/arm64; do
  suffix="${platform#linux/}"
  image="json-handle-cli-benzhi:${suffix}"
  docker build --platform "${platform}" -f benzhi.Dockerfile -t "${image}" .
  docker run --rm --platform "${platform}" "${image}" go build ./...
  docker run --rm --platform "${platform}" "${image}"
done
