#!/usr/bin/env python3
"""Shared local/Paseo/CI checks. Never suppress an existing failure."""
import argparse
import os
from pathlib import Path
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[1]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--integrations", action="store_true", help="also run both integration template gates")
    parser.add_argument("--vet", action="store_true", help="include go vet (existing baseline issues are reported)")
    args = parser.parse_args()
    base = os.environ.get("N9E_VERIFY_BASE")
    if not base:
        result = subprocess.run(["git", "merge-base", "HEAD", "main"], cwd=ROOT, capture_output=True, text=True)
        base = result.stdout.strip() if result.returncode == 0 else "HEAD"
    changed = subprocess.check_output(["git", "diff", "--name-only", "--diff-filter=ACMR", "-z", base], cwd=ROOT)
    changed += subprocess.check_output(["git", "ls-files", "--others", "--exclude-standard", "-z"], cwd=ROOT)
    files = {os.fsdecode(p) for p in changed.split(b"\0") if p}
    go_files = sorted(p for p in files if p.endswith(".go") and not p.startswith("front/statik/") and (ROOT / p).is_file())
    failures = []
    if go_files:
        result = subprocess.run(["gofmt", "-l", *go_files], cwd=ROOT, capture_output=True, text=True)
        if result.returncode or result.stdout:
            print("gofmt failed:\n" + result.stdout + result.stderr, flush=True)
            failures.append("format")
    checks = [("preview tests", ["python3", "-B", "-m", "unittest", "discover", "-s", "scripts", "-p", "test_*.py"]),
              ("build", ["go", "build", "./..."]), ("tests", ["go", "test", "./..."])]
    if args.vet:
        checks.append(("vet", ["go", "vet", "./..."]))
    if args.integrations or any(p.startswith(("integrations/", "center/integration/", "cmd/integrations-")) for p in files):
        checks.extend([("integration actions", ["go", "run", "./cmd/integrations-alert-action", "check"]),
                       ("integration i18n", ["go", "run", "./cmd/integrations-i18n", "check", "-scope", "all"])])
    for label, command in checks:
        print(f"\nChecking {label}: {' '.join(command)}", flush=True)
        if subprocess.run(command, cwd=ROOT).returncode:
            failures.append(label)
    if failures:
        print("\nFailed: " + ", ".join(failures), file=sys.stderr)
        return 1
    print("\nAll requested checks passed.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
