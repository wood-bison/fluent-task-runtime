#!/usr/bin/env bash
set -Eeuo pipefail

output=/output/results.json
set +e
bash /hidden-tests/tests/check.sh
status=$?
set -e
if [[ "$status" -eq 0 ]]; then state=pass; else state=fail; fi
if [[ "$state" == "fail" ]]; then
  message=', "message": "The submitted Java solution failed one or more checks."'
else
  message=''
fi
cat > "$output" <<EOF
{
  "version": 2,
  "status": "$state",
  "tests": [
    {"name": "allows requests inside the configured burst", "status": "$state"$message},
    {"name": "refills tokens over elapsed time", "status": "$state"$message},
    {"name": "isolates independent keys", "status": "$state"$message},
    {"name": "rejects requests after the bucket is empty", "status": "$state"$message}
  ]
}
EOF
