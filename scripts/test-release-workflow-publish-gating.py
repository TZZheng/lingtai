#!/usr/bin/env python3
"""Static assertions for the source-only tag release workflow.

The tag workflow must create a GitHub source release and update the Homebrew
source-build formula. It must not build, package, upload, or mirror prebuilt
binary/bundle artifacts.
"""
from __future__ import annotations

import sys
from pathlib import Path

try:
    import yaml
except ModuleNotFoundError:
    print("SKIP: PyYAML not available in this environment", file=sys.stderr)
    raise SystemExit(0)

REPO_ROOT = Path(__file__).resolve().parents[1]
WORKFLOW_PATH = REPO_ROOT / ".github" / "workflows" / "release.yml"
FAILURES: list[str] = []


def check(condition: bool, message: str) -> None:
    if not condition:
        FAILURES.append(message)


def find_step(job: dict, name_substring: str) -> dict | None:
    for step in job.get("steps", []):
        if name_substring.lower() in step.get("name", "").lower():
            return step
    return None


def main() -> int:
    data = yaml.safe_load(WORKFLOW_PATH.read_text())
    on = data.get("on") or data.get(True)  # YAML 1.1 bareword quirk guard
    check("push" in on and "tags" in on["push"], "release.yml must trigger on tag pushes")
    check(any("v*" in tag for tag in on["push"]["tags"]), "release.yml must trigger on v* tags")

    jobs = data.get("jobs", {})
    check(set(jobs) == {"source-release", "update-homebrew"},
          f"release jobs must be source-release + update-homebrew only, got {sorted(jobs)}")

    source = jobs.get("source-release", {})
    create = find_step(source, "create github source release")
    check(create is not None, "source-release must have a GitHub source release step")
    if create:
        script = create.get("run", "")
        check("gh release create" in script, "source-release must create the GitHub release")
        check("--verify-tag" in script, "source-release must verify the pushed tag")
        check("gh release upload" not in script, "source-release must not upload binary assets")

    homebrew = jobs.get("update-homebrew", {})
    checksum = find_step(homebrew, "compute source tarball checksum")
    formula = find_step(homebrew, "write formula")
    check(checksum is not None, "update-homebrew must checksum the GitHub source tarball")
    check(formula is not None, "update-homebrew must write the source-build formula")
    if formula:
        script = formula.get("run", "")
        check("archive/refs/tags/${TAG}.tar.gz" in script,
              "Homebrew formula must build from the GitHub tag source archive")
        check('depends_on "go" => :build' in script,
              "Homebrew formula must retain source-build Go dependency")

    text = WORKFLOW_PATH.read_text()
    forbidden = (
        "release-assets", "publish-bundle", "gh release upload",
        "lingtai-bundle-manifest.json", "publish_bundle_to_gitee.sh",
        "sync_gitee_mirror.sh", "GITEE_ACCESS_TOKEN",
    )
    for needle in forbidden:
        check(needle not in text, f"source-only workflow must not contain {needle!r}")

    if FAILURES:
        print("FAILED source-only release workflow checks:", file=sys.stderr)
        for failure in FAILURES:
            print(f"  - {failure}", file=sys.stderr)
        return 1
    print("OK: source-only GitHub release + Homebrew workflow, no prebuilt/bundle publication")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
