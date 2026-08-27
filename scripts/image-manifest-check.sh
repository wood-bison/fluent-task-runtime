#!/usr/bin/env bash
set -Eeuo pipefail

# Verify that every executable task points at the same immutable image
# reference declared by task-images/manifest.json. `--static` validates the
# repository contract without requiring Docker; the default additionally
# proves that every digest is present in the local daemon.

RUNTIME_ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd -P)"
MANIFEST="$RUNTIME_ROOT/task-images/manifest.json"
static=false
if [[ "${1:-}" == "--static" ]]; then
  static=true
elif [[ "${1:-}" != "" ]]; then
  echo "Usage: $0 [--static]" >&2
  exit 2
fi

command -v jq >/dev/null 2>&1 || { echo 'Missing command: jq' >&2; exit 1; }
test -s "$MANIFEST" || { echo "Missing image manifest: $MANIFEST" >&2; exit 1; }

digest_ref='@sha256:[0-9a-f]{64}$'
mapfile -t expected_images < <(jq -r '.images[]?.image // empty' "$MANIFEST")
if (( ${#expected_images[@]} == 0 )); then
  echo 'Image manifest has no images.' >&2
  exit 1
fi

declare -A expected
for image in "${expected_images[@]}"; do
  if [[ ! "$image" =~ $digest_ref ]]; then
    echo "Image manifest contains a mutable or malformed reference: $image" >&2
    exit 1
  fi
  expected["$image"]=1
done

if [[ "$static" != true ]]; then
  command -v docker >/dev/null 2>&1 || { echo 'Missing command: docker' >&2; exit 1; }
  docker info >/dev/null 2>&1 || { echo 'Docker daemon is not reachable.' >&2; exit 1; }
  for image in "${expected_images[@]}"; do
    if ! docker image inspect "$image" >/dev/null 2>&1; then
      echo "Pinned task image is not available locally: $image" >&2
      exit 1
    fi
  done
fi

task_count=0
while IFS= read -r descriptor; do
  task_count=$((task_count + 1))
  image="$(jq -r '.image // empty' "$descriptor")"
  if [[ -z "$image" || -z "${expected[$image]+present}" ]]; then
    echo "Task descriptor does not use a declared immutable image: $descriptor -> ${image:-<missing>}" >&2
    exit 1
  fi
done < <(find "$RUNTIME_ROOT/tasks" -mindepth 2 -maxdepth 2 -type f -name task.json -print | sort)

if rg -n 'fluent-runtime-task-(node|postgres|dotnet|go|java):[0-9]+' \
  "$RUNTIME_ROOT/task-images/manifest.json" "$RUNTIME_ROOT/tasks" "$RUNTIME_ROOT/internal/engine/catalogue.go" >/dev/null; then
  echo 'Mutable task image tag found in the executable catalogue.' >&2
  exit 1
fi

mode='static'
[[ "$static" == true ]] || mode='daemon'
printf 'task image manifest: %s immutable references verified (%d task descriptors, %s mode)\n' \
  "${#expected_images[@]}" "$task_count" "$mode"
