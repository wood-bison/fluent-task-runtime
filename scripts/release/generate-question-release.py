#!/usr/bin/env python3
"""Pin a Task Runtime release to the currently published Question Brain release.

This is deliberately a small release tool rather than runtime code.  It reads
the immutable task bindings from an existing manifest, resolves each stable
key against Question Brain's answer-free ``/v1/release`` identity manifest,
and writes a new manifest.  Capability keys are read from the authored task
descriptor when available so the executable station crosswalk is part of the
same reviewed source as the task brief.
The old manifest remains immutable history.
"""

from __future__ import annotations

import argparse
import json
import sys
import urllib.parse
import urllib.request
from pathlib import Path


def fetch_json(url: str) -> dict:
    request = urllib.request.Request(url, headers={"Accept": "application/json"})
    with urllib.request.urlopen(request, timeout=30) as response:
        return json.load(response)


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--source", required=True, type=Path, help="previous task release manifest")
    parser.add_argument("--question-brain", required=True, help="Question Brain base URL")
    parser.add_argument("--workspace", default="fluent-interview")
    parser.add_argument(
        "--tasks-root",
        default=Path("tasks"),
        type=Path,
        help="authored task descriptor root used for capability keys",
    )
    parser.add_argument("--release-id", required=True, help="new immutable runtime release id")
    parser.add_argument("--output", required=True, type=Path)
    args = parser.parse_args()

    source = json.loads(args.source.read_text(encoding="utf-8"))
    if source.get("contractVersion") != "fluent-task-runtime.task-release.v1":
        raise SystemExit("source manifest has an unsupported contractVersion")

    query = urllib.parse.urlencode({"workspace": args.workspace})
    url = args.question_brain.rstrip("/") + "/v1/release?" + query
    release = fetch_json(url)
    question_release_id = release.get("release_id", "")
    if not (question_release_id.startswith("question-release-") and len(question_release_id) == len("question-release-") + 16):
        raise SystemExit(f"Question Brain returned an invalid release_id: {question_release_id!r}")

    identities = {}
    for item in release.get("items", []):
        stable_key = item.get("stable_key", "")
        if stable_key:
            identities[stable_key] = item

    tasks = []
    missing = []
    for task in source.get("tasks", []):
        descriptor_path = args.tasks_root / task["taskId"] / "task.json"
        descriptor = {}
        if descriptor_path.is_file():
            descriptor = json.loads(descriptor_path.read_text(encoding="utf-8"))
        copied = {
            "taskId": task["taskId"],
            "revision": task["revision"],
            "questionBindings": [],
            "capabilityKeys": list(
                descriptor.get("capabilityKeys", task.get("capabilityKeys", []))
            ),
        }
        for binding in task.get("questionBindings", []):
            stable_key = binding.get("stableKey", "")
            identity = identities.get(stable_key)
            if identity is None:
                missing.append(stable_key)
                continue
            copied["questionBindings"].append(
                {
                    "stableKey": stable_key,
                    "revisionId": identity["revision_id"],
                    "contentHash": identity["content_hash"],
                }
            )
        tasks.append(copied)

    if missing:
        raise SystemExit("Question Brain release is missing stable keys: " + ", ".join(sorted(set(missing))))

    output = {
        "contractVersion": "fluent-task-runtime.task-release.v1",
        "releaseId": args.release_id,
        "questionReleaseId": question_release_id,
        "tasks": tasks,
    }
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(output, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
    print(json.dumps({"runtimeReleaseId": args.release_id, "questionReleaseId": question_release_id, "tasks": len(tasks), "output": str(args.output)}, indent=2))
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except urllib.error.HTTPError as error:
        print(f"Question Brain request failed: HTTP {error.code}", file=sys.stderr)
        raise
