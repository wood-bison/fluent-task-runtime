#!/usr/bin/env bash
set -Eeuo pipefail

output=/output/results.json
set +e
bash /hidden-tests/tests/check.sh
status=$?
set -e
if [[ "$status" -eq 0 ]]; then state=pass; else state=fail; fi
cat > "$output" <<EOF
{
  "version": 2,
  "status": "$state",
  "tests": [
    {"name": "allows requests inside the configured burst", "status": "$state"},
    {"name": "refills tokens over elapsed time", "status": "$state"},
    {"name": "isolates independent keys", "status": "$state"},
    {"name": "rejects requests after the bucket is empty", "status": "$state"}
  ]
}
EOF
