#!/usr/bin/env python3
"""Build a fail-closed Task Runtime release join from both release APIs.

Question Brain supplies the exact question revision/hash identities and the
active canonical capability snapshot. Task Runtime supplies the released
TaskFamily/revision join. The generated v3 manifest is the only runnable join;
the old manifests remain immutable history and are never edited.
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
    parser.add_argument(
        "--runtime-api",
        default="http://127.0.0.1:48227",
        help="Task Runtime base URL used to pin the TaskFamily release",
    )
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
    if source.get("contractVersion") not in {
        "fluent-task-runtime.task-release.v1",
        "fluent-task-runtime.task-release.v2",
    }:
        raise SystemExit("source manifest must be a v1/v2 runtime release history")

    query = urllib.parse.urlencode({"workspace": args.workspace})
    url = args.question_brain.rstrip("/") + "/v1/release?" + query
    release = fetch_json(url)
    if release.get("contract_version") != "question-brain.release.v1":
        raise SystemExit("Question Brain returned an unsupported release contract")
    if release.get("workspace_key") != args.workspace:
        raise SystemExit(
            "Question Brain returned the wrong workspace: "
            + repr(release.get("workspace_key"))
        )
    question_release_id = release.get("release_id", "")
    if not (question_release_id.startswith("question-release-") and len(question_release_id) == len("question-release-") + 16):
        raise SystemExit(f"Question Brain returned an invalid release_id: {question_release_id!r}")
    question_snapshot_id = release.get("source_snapshot_id", "")
    capability_registry_release_id = release.get("capability_registry_release_id", "")
    capability_binding_release_id = release.get("capability_binding_release_id", "")
    if not capability_registry_release_id or not capability_binding_release_id:
        raise SystemExit("Question Brain did not expose an active capability release")
    capability_keys = sorted(
        key for key in release.get("capability_keys", []) if isinstance(key, str)
    )
    if not capability_keys:
        raise SystemExit("Question Brain returned an empty active capability snapshot")

    family_release = fetch_json(args.runtime_api.rstrip("/") + "/v1/task-families")
    if family_release.get("contractVersion") != "fluent-task-runtime.task-families.v1":
        raise SystemExit("Task Runtime returned an unsupported TaskFamily contract")
    task_family_release_id = family_release.get("releaseId", "")
    if not task_family_release_id:
        raise SystemExit("Task Runtime returned no TaskFamily release ID")
    family_revisions = {
        (revision.get("taskId"), revision.get("revision")): family.get("key")
        for family in family_release.get("families", [])
        for revision in family.get("revisions", [])
    }
    families_by_key = {
        family.get("key"): family for family in family_release.get("families", [])
    }
    family_keys = set(families_by_key)

    identities = {}
    for item in release.get("items", []):
        stable_key = item.get("stable_key", "")
        if stable_key:
            identities[stable_key] = item

    tasks = []
    missing = []
    invalid = []
    for task in source.get("tasks", []):
        descriptor_path = args.tasks_root / task["taskId"] / "task.json"
        descriptor = {}
        if descriptor_path.is_file():
            descriptor = json.loads(descriptor_path.read_text(encoding="utf-8"))
        task_family_key = task.get("taskFamilyKey", "")
        if not task_family_key or task_family_key not in family_keys:
            invalid.append(f"{task.get('taskId')}@{task.get('revision')}: missing TaskFamily")
        if family_revisions.get((task.get("taskId"), task.get("revision"))) != task_family_key:
            invalid.append(f"{task.get('taskId')}@{task.get('revision')}: TaskFamily revision mismatch")
        family = families_by_key.get(task_family_key, {})
        descriptor_capabilities = family.get("capabilityKeys")
        if not isinstance(descriptor_capabilities, list):
            descriptor_capabilities = descriptor.get("capabilityKeys")
        if not isinstance(descriptor_capabilities, list):
            descriptor_capabilities = task.get("capabilityKeys", [])
        task_capability_keys = sorted({key for key in descriptor_capabilities if isinstance(key, str)})
        for capability_key in task_capability_keys:
            if capability_key not in capability_keys:
                invalid.append(
                    f"{task.get('taskId')}@{task.get('revision')}: capability {capability_key!r} is not active in {capability_registry_release_id}"
                )
        copied = {
            "taskId": task["taskId"],
            "revision": task["revision"],
            "taskFamilyKey": task_family_key,
            "questionBindings": [],
            "capabilityKeys": task_capability_keys,
        }
        for binding in task.get("questionBindings", []):
            stable_key = binding.get("stableKey", "")
            identity = identities.get(stable_key)
            if identity is None:
                missing.append(stable_key)
                continue
            if identity.get("status") != "published" or identity.get("content_kind") != "production":
                invalid.append(f"{stable_key}: is not a published production QuestionCard")
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
    if invalid:
        raise SystemExit("release join validation failed:\n- " + "\n- ".join(sorted(set(invalid))))
    for task in tasks:
        if not task["questionBindings"] and not task["capabilityKeys"]:
            raise SystemExit(
                f"{task['taskId']}@{task['revision']} has neither questionBindings nor capabilityKeys"
            )

    output = {
        "contractVersion": "fluent-task-runtime.task-release.v3",
        "releaseId": args.release_id,
        "workspaceKey": args.workspace,
        "questionBrainContractVersion": release["contract_version"],
        "questionReleaseId": question_release_id,
        "questionSourceSnapshotId": question_snapshot_id,
        "capabilityBindingReleaseId": capability_binding_release_id,
        "capabilityRegistryReleaseId": capability_registry_release_id,
        "capabilityKeys": capability_keys,
        "taskFamilyReleaseId": task_family_release_id,
        "tasks": tasks,
    }
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(output, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
    print(
        json.dumps(
            {
                "runtimeReleaseId": args.release_id,
                "questionReleaseId": question_release_id,
                "capabilityRegistryReleaseId": capability_registry_release_id,
                "capabilityBindingReleaseId": capability_binding_release_id,
                "taskFamilyReleaseId": task_family_release_id,
                "tasks": len(tasks),
                "output": str(args.output),
            },
            indent=2,
        )
    )
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except urllib.error.HTTPError as error:
        print(f"Question Brain request failed: HTTP {error.code}", file=sys.stderr)
        raise
