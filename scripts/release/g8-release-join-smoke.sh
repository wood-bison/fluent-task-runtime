#!/usr/bin/env bash
set -euo pipefail

runtime_url="${RUNTIME_API:-http://127.0.0.1:48227}"
brain_url="${QUESTION_BRAIN_API:-http://127.0.0.1:48127}"
workspace="${QUESTION_BRAIN_WORKSPACE:-fluent-interview}"
tmp_manifest="$(mktemp -t fluent-task-runtime-g8.XXXXXX.json)"
trap 'rm -f "$tmp_manifest"' EXIT

source_manifest="${RUNTIME_RELEASE_SOURCE:-releases/task-release-2026-08-25-qb-d00a1493-g9.json}"
test -s "${source_manifest}"
expected_task_count="$(jq '.tasks | length' "${source_manifest}")"
test "${expected_task_count}" -gt 0

summary="$(curl -fsS "$runtime_url/v1/tasks/summary")"
printf '%s\n' "$summary" | jq -e \
  --arg workspace "$workspace" \
  --argjson expectedTaskCount "$expected_task_count" \
  '(.contractVersion == "fluent-task-runtime.task-summary.v1") and
   (.bindingState == "manifest-loaded") and (.runnable == true) and
   (.runtimeReleaseId | startswith("runtime-task-release-")) and
   (.questionReleaseId | startswith("question-release-")) and
   (.questionSourceSnapshotId != null) and
   (.capabilityBindingReleaseId != null) and
   (.capabilityRegistryReleaseId != null) and
   (.taskFamilyReleaseId != null) and
   ((.tasks | length) == $expectedTaskCount) and
   ([.tasks[] | select(.status == "released") | (.taskFamilyKey != null and (.questionBindings | type == "array") and (.capabilityKeys | type == "array"))] | all)' \
  >/dev/null

python3 scripts/release/generate-question-release.py \
  --source "${source_manifest}" \
  --question-brain "$brain_url" \
  --runtime-api "$runtime_url" \
  --workspace "$workspace" \
  --tasks-root tasks \
  --release-id runtime-task-release-smoke-g8 \
  --output "$tmp_manifest" \
  >/dev/null

jq -e \
  --argjson expectedTaskCount "$expected_task_count" \
  '(.contractVersion == "fluent-task-runtime.task-release.v3") and
   (.workspaceKey == "fluent-interview") and
   ((.tasks | length) == $expectedTaskCount) and
   ((.capabilityKeys | length) > 0) and
   ([.tasks[] | (.taskFamilyKey != null and (.questionBindings | type == "array") and (.capabilityKeys | type == "array") and (has("questionKeys") | not))] | all)' \
  "$tmp_manifest" >/dev/null

printf 'G8 release join smoke passed: %s tasks, runtime=%s, question=%s, task-families=%s\n' \
  "$(jq '.tasks | length' "$tmp_manifest")" \
  "$(jq -r '.releaseId' "$tmp_manifest")" \
  "$(jq -r '.questionReleaseId' "$tmp_manifest")" \
  "$(jq -r '.taskFamilyReleaseId' "$tmp_manifest")"
